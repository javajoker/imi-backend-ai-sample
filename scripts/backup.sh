#!/bin/bash

# IP Marketplace Database Backup Script

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${PROJECT_ROOT}/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Load environment variables
if [ -f "${PROJECT_ROOT}/.env" ]; then
    source "${PROJECT_ROOT}/.env"
fi

# Default values if not set in .env
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-ip_marketplace}
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-30}

echo "🗄️  Starting database backup..."

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Function to backup database
backup_database() {
    local backup_file="${BACKUP_DIR}/db_backup_${DATE}.sql"
    local backup_file_compressed="${backup_file}.gz"
    
    echo "Creating database backup: $(basename "$backup_file_compressed")"
    
    # Create SQL dump
    PGPASSWORD="$DB_PASSWORD" pg_dump \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --verbose \
        --no-password \
        --format=plain \
        --no-owner \
        --no-privileges > "$backup_file"
    
    if [ $? -eq 0 ]; then
        echo "✅ Database dump created successfully"
        
        # Compress backup
        gzip "$backup_file"
        echo "✅ Backup compressed: $(basename "$backup_file_compressed")"
        
        # Get file size
        local file_size=$(du -h "$backup_file_compressed" | cut -f1)
        echo "📊 Backup size: $file_size"
        
        return 0
    else
        echo "❌ Database backup failed"
        [ -f "$backup_file" ] && rm "$backup_file"
        return 1
    fi
}

# Function to backup uploaded files
backup_files() {
    local uploads_dir="${PROJECT_ROOT}/uploads"
    local files_backup="${BACKUP_DIR}/uploads_backup_${DATE}.tar.gz"
    
    if [ -d "$uploads_dir" ]; then
        echo "Creating files backup..."
        
        tar -czf "$files_backup" -C "$PROJECT_ROOT" uploads/
        
        if [ $? -eq 0 ]; then
            local file_size=$(du -h "$files_backup" | cut -f1)
            echo "✅ Files backup created: $(basename "$files_backup") ($file_size)"
        else
            echo "❌ Files backup failed"
            [ -f "$files_backup" ] && rm "$files_backup"
        fi
    else
        echo "⚠️  Uploads directory not found, skipping files backup"
    fi
}

# Function to cleanup old backups
cleanup_old_backups() {
    echo "🧹 Cleaning up backups older than $BACKUP_RETENTION_DAYS days..."
    
    find "$BACKUP_DIR" -name "db_backup_*.sql.gz" -mtime +$BACKUP_RETENTION_DAYS -delete
    find "$BACKUP_DIR" -name "uploads_backup_*.tar.gz" -mtime +$BACKUP_RETENTION_DAYS -delete
    
    echo "✅ Cleanup completed"
}

# Function to upload to S3 (if configured)
upload_to_s3() {
    if [ -n "$AWS_S3_BACKUP_BUCKET" ] && command -v aws &> /dev/null; then
        echo "📤 Uploading backups to S3..."
        
        aws s3 cp "$BACKUP_DIR/" "s3://$AWS_S3_BACKUP_BUCKET/backups/" \
            --recursive \
            --exclude "*" \
            --include "*_${DATE}.*"
        
        if [ $? -eq 0 ]; then
            echo "✅ Backups uploaded to S3"
        else
            echo "❌ S3 upload failed"
        fi
    fi
}

# Main execution
main() {
    echo "Starting backup process at $(date)"
    echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
    echo "Backup directory: $BACKUP_DIR"
    echo ""
    
    # Perform backups
    if backup_database; then
        backup_files
        cleanup_old_backups
        upload_to_s3
        
        echo ""
        echo "🎉 Backup completed successfully at $(date)"
        echo "📁 Backup location: $BACKUP_DIR"
        
        # List recent backups
        echo ""
        echo "Recent backups:"
        ls -lah "$BACKUP_DIR" | grep "${DATE:0:8}" || echo "No backups found for today"
    else
        echo ""
        echo "❌ Backup failed at $(date)"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    if ! command -v pg_dump &> /dev/null; then
        echo "❌ pg_dump not found. Please install PostgreSQL client tools."
        exit 1
    fi
    
    if [ -z "$DB_PASSWORD" ]; then
        echo "❌ DB_PASSWORD not set. Please configure your environment variables."
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --db-only      Backup database only"
    echo "  --files-only   Backup files only"
    echo "  --no-cleanup   Skip cleanup of old backups"
    echo ""
    echo "Environment variables:"
    echo "  DB_HOST, DB_PORT, DB_USER, DB_NAME, DB_PASSWORD"
    echo "  BACKUP_RETENTION_DAYS (default: 30)"
    echo "  AWS_S3_BACKUP_BUCKET (optional)"
}

# Parse command line arguments
DB_ONLY=false
FILES_ONLY=false
NO_CLEANUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        --db-only)
            DB_ONLY=true
            shift
            ;;
        --files-only)
            FILES_ONLY=true
            shift
            ;;
        --no-cleanup)
            NO_CLEANUP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Execute based on options
check_dependencies

if [ "$DB_ONLY" = true ]; then
    backup_database
elif [ "$FILES_ONLY" = true ]; then
    backup_files
else
    main
fi
