// internal/middleware/i18n.go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get language from header
		lang := c.GetHeader("Accept-Language")

		// Parse language preference
		if lang != "" {
			// Handle cases like "zh-TW,zh;q=0.9,en;q=0.8"
			langs := strings.Split(lang, ",")
			if len(langs) > 0 {
				firstLang := strings.TrimSpace(strings.Split(langs[0], ";")[0])
				// Convert common language codes
				switch firstLang {
				case "zh-TW", "zh-Hant", "zh_TW":
					lang = "zh_TW"
				case "zh-CN", "zh-Hans", "zh_CN":
					lang = "zh_CN"
				case "en", "en-US", "en-GB":
					lang = "en"
				default:
					lang = "en" // Default to English
				}
			}
		} else {
			lang = "en" // Default language
		}

		// Set language in context
		c.Set("lang", lang)
		c.Next()
	}
}
