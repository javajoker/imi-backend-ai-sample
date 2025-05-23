// internal/i18n/i18n.go
package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

type I18n struct {
	mu           sync.RWMutex
	translations map[string]map[string]string
	defaultLang  string
}

var instance *I18n
var once sync.Once

func Initialize() error {
	var err error
	once.Do(func() {
		instance = &I18n{
			translations: make(map[string]map[string]string),
			defaultLang:  "en",
		}
		err = instance.LoadTranslations("./internal/i18n/locales")
	})
	return err
}

func (i *I18n) LoadTranslations(localesPath string) error {
	localeFiles := []string{"en.json", "zh_TW.json"}

	for _, file := range localeFiles {
		lang := strings.TrimSuffix(file, ".json")
		filePath := filepath.Join(localesPath, file)

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read locale file %s: %w", filePath, err)
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("failed to unmarshal locale file %s: %w", filePath, err)
		}

		i.mu.Lock()
		i.translations[lang] = translations
		i.mu.Unlock()
	}

	return nil
}

func (i *I18n) T(lang, key string, args ...interface{}) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Try to get translation for requested language
	if translations, exists := i.translations[lang]; exists {
		if text, exists := translations[key]; exists {
			if len(args) > 0 {
				return fmt.Sprintf(text, args...)
			}
			return text
		}
	}

	// Fallback to default language
	if lang != i.defaultLang {
		if translations, exists := i.translations[i.defaultLang]; exists {
			if text, exists := translations[key]; exists {
				if len(args) > 0 {
					return fmt.Sprintf(text, args...)
				}
				return text
			}
		}
	}

	// Return key if no translation found
	return key
}

// Global functions
func T(lang, key string, args ...interface{}) string {
	if instance != nil {
		return instance.T(lang, key, args...)
	}
	return key
}

func GetSupportedLanguages() []string {
	if instance == nil {
		return []string{"en"}
	}

	instance.mu.RLock()
	defer instance.mu.RUnlock()

	langs := make([]string, 0, len(instance.translations))
	for lang := range instance.translations {
		langs = append(langs, lang)
	}
	return langs
}
