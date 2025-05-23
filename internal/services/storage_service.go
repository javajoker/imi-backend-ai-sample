// internal/services/storage_service.go
package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/config"
)

type StorageService struct {
	s3Client *s3.S3
	config   *config.Config
}

type UploadResult struct {
	URL      string `json:"url"`
	Key      string `json:"key"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type UploadOptions struct {
	Folder       string
	MaxSize      int64 // in bytes
	AllowedTypes []string
	IsPublic     bool
}

func NewStorageService(config *config.Config) (*StorageService, error) {
	if config.AWS.AccessKeyID == "" {
		// Return service without S3 for local development
		return &StorageService{config: config}, nil
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWS.Region),
		Credentials: credentials.NewStaticCredentials(
			config.AWS.AccessKeyID,
			config.AWS.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &StorageService{
		s3Client: s3.New(sess),
		config:   config,
	}, nil
}

func (s *StorageService) UploadFile(file multipart.File, header *multipart.FileHeader, options UploadOptions) (*UploadResult, error) {
	// Validate file size
	if options.MaxSize > 0 && header.Size > options.MaxSize {
		return nil, fmt.Errorf("file size %d bytes exceeds maximum allowed size %d bytes", header.Size, options.MaxSize)
	}

	// Validate file type
	if len(options.AllowedTypes) > 0 {
		fileExt := strings.ToLower(filepath.Ext(header.Filename))
		allowed := false
		for _, allowedType := range options.AllowedTypes {
			if fileExt == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("file type %s is not allowed", fileExt)
		}
	}

	// Generate unique filename
	filename := s.generateFileName(header.Filename, options.Folder)

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Upload to S3 or local storage
	if s.s3Client != nil {
		return s.uploadToS3(fileBytes, filename, header.Header.Get("Content-Type"), options.IsPublic)
	}

	return s.uploadToLocal(fileBytes, filename, header.Header.Get("Content-Type"))
}

func (s *StorageService) uploadToS3(fileBytes []byte, key, contentType string, isPublic bool) (*UploadResult, error) {
	// Prepare S3 upload parameters
	params := &s3.PutObjectInput{
		Bucket:        aws.String(s.config.AWS.S3Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(fileBytes),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}

	if isPublic {
		params.ACL = aws.String("public-read")
	}

	// Upload to S3
	_, err := s.s3Client.PutObject(params)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate URL
	url := s.getS3URL(key)

	return &UploadResult{
		URL:      url,
		Key:      key,
		Size:     int64(len(fileBytes)),
		MimeType: contentType,
	}, nil
}

func (s *StorageService) uploadToLocal(fileBytes []byte, filename, contentType string) (*UploadResult, error) {
	// For local development, we'll simulate file storage
	// In a real implementation, you'd save to local filesystem

	url := fmt.Sprintf("http://localhost:8080/uploads/%s", filename)

	return &UploadResult{
		URL:      url,
		Key:      filename,
		Size:     int64(len(fileBytes)),
		MimeType: contentType,
	}, nil
}

func (s *StorageService) DeleteFile(key string) error {
	if s.s3Client == nil {
		// Local development - just log
		fmt.Printf("File would be deleted: %s\n", key)
		return nil
	}

	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.config.AWS.S3Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (s *StorageService) GeneratePresignedURL(key string, expiration time.Duration) (string, error) {
	if s.s3Client == nil {
		return "", fmt.Errorf("S3 client not configured")
	}

	req, _ := s.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.config.AWS.S3Bucket),
		Key:    aws.String(key),
	})

	url, err := req.Presign(expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

func (s *StorageService) GetDefaultUploadOptions(category string) UploadOptions {
	switch category {
	case "ip_assets":
		return UploadOptions{
			Folder:       "ip-assets",
			MaxSize:      50 * 1024 * 1024, // 50MB
			AllowedTypes: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".mp4", ".mp3", ".zip"},
			IsPublic:     true,
		}
	case "products":
		return UploadOptions{
			Folder:       "products",
			MaxSize:      10 * 1024 * 1024, // 10MB
			AllowedTypes: []string{".jpg", ".jpeg", ".png", ".gif"},
			IsPublic:     true,
		}
	case "avatars":
		return UploadOptions{
			Folder:       "avatars",
			MaxSize:      2 * 1024 * 1024, // 2MB
			AllowedTypes: []string{".jpg", ".jpeg", ".png"},
			IsPublic:     true,
		}
	default:
		return UploadOptions{
			Folder:       "general",
			MaxSize:      5 * 1024 * 1024, // 5MB
			AllowedTypes: []string{".jpg", ".jpeg", ".png", ".pdf"},
			IsPublic:     false,
		}
	}
}

func (s *StorageService) generateFileName(originalName, folder string) string {
	// Generate UUID for uniqueness
	id := uuid.New()

	// Get file extension
	ext := filepath.Ext(originalName)

	// Create filename with timestamp and UUID
	timestamp := time.Now().Format("20060102")
	filename := fmt.Sprintf("%s_%s%s", timestamp, id.String()[:8], ext)

	if folder != "" {
		return fmt.Sprintf("%s/%s", folder, filename)
	}

	return filename
}

func (s *StorageService) getS3URL(key string) string {
	if s.config.AWS.CloudFrontURL != "" {
		return fmt.Sprintf("%s/%s", s.config.AWS.CloudFrontURL, key)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		s.config.AWS.S3Bucket, s.config.AWS.Region, key)
}

func (s *StorageService) ValidateImage(file multipart.File) error {
	// Read first few bytes to check file signature
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Check for common image signatures
	if !s.isValidImageType(buffer) {
		return fmt.Errorf("invalid image file")
	}

	return nil
}

func (s *StorageService) isValidImageType(buffer []byte) bool {
	// Check for JPEG
	if len(buffer) >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
		return true
	}

	// Check for PNG
	if len(buffer) >= 8 && buffer[0] == 0x89 && buffer[1] == 0x50 && buffer[2] == 0x4E && buffer[3] == 0x47 {
		return true
	}

	// Check for GIF
	if len(buffer) >= 6 && string(buffer[0:6]) == "GIF87a" || string(buffer[0:6]) == "GIF89a" {
		return true
	}

	return false
}
