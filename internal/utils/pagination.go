// internal/utils/pagination.go
package utils

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaginationParams struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	Sort     string `json:"sort"`
	Order    string `json:"order"`
	Search   string `json:"search"`
	Category string `json:"category"`
}

type PaginationResult struct {
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
	Data       interface{} `json:"data"`
}

func GetPaginationParams(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sort := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")
	search := c.Query("search")
	category := c.Query("category")

	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return PaginationParams{
		Page:     page,
		Limit:    limit,
		Sort:     sort,
		Order:    order,
		Search:   search,
		Category: category,
	}
}

func ApplyPagination(db *gorm.DB, params PaginationParams) *gorm.DB {
	offset := (params.Page - 1) * params.Limit
	return db.Offset(offset).Limit(params.Limit)
}

func ApplySort(db *gorm.DB, params PaginationParams, allowedSortFields []string) *gorm.DB {
	// Validate sort field
	sortField := params.Sort
	validSort := false
	for _, field := range allowedSortFields {
		if field == sortField {
			validSort = true
			break
		}
	}

	if !validSort {
		sortField = "created_at"
	}

	return db.Order(sortField + " " + params.Order)
}

func CreatePaginationResult(data interface{}, total int64, params PaginationParams) PaginationResult {
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return PaginationResult{
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
		Data:       data,
	}
}

func SetPaginationHeaders(c *gin.Context, result PaginationResult) {
	c.Header("X-Total-Count", strconv.FormatInt(result.Total, 10))
	c.Header("X-Page", strconv.Itoa(result.Page))
	c.Header("X-Per-Page", strconv.Itoa(result.Limit))
	c.Header("X-Total-Pages", strconv.Itoa(result.TotalPages))
}
