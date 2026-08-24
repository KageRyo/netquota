// Package i18n provides NetQuota's application-level translations.
//
// Fyne's translation helpers select a language from the operating system. The
// application has its own persisted language preference, so user-facing copy
// is resolved through this package instead.
package i18n

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"text/template"
)

type Language string

const (
	English            Language = "en"
	TraditionalChinese Language = "zh-Hant"
	Japanese           Language = "ja"
)

var supportedLanguages = []Language{English, TraditionalChinese, Japanese}

//go:embed translation/*.json
var translations embed.FS

var catalogs map[Language]map[string]string

func init() {
	catalogs = make(map[Language]map[string]string, len(supportedLanguages))
	for _, language := range supportedLanguages {
		contents, err := translations.ReadFile("translation/" + string(language) + ".json")
		if err != nil {
			panic(fmt.Sprintf("read %s translations: %v", language, err))
		}
		catalog := make(map[string]string)
		if err := json.Unmarshal(contents, &catalog); err != nil {
			panic(fmt.Sprintf("parse %s translations: %v", language, err))
		}
		catalogs[language] = catalog
	}
	if err := ValidateCatalogs(); err != nil {
		panic(err)
	}
}

func SupportedLanguages() []Language {
	return append([]Language(nil), supportedLanguages...)
}

func (language Language) Valid() bool {
	_, ok := catalogs[language]
	return ok
}

func Normalize(language Language) Language {
	if language.Valid() {
		return language
	}
	return English
}

func DisplayName(language Language) string {
	switch Normalize(language) {
	case TraditionalChinese:
		return "正體中文"
	case Japanese:
		return "日本語"
	default:
		return "English"
	}
}

func ParseDisplayName(name string) (Language, bool) {
	for _, language := range supportedLanguages {
		if name == DisplayName(language) {
			return language, true
		}
	}
	return "", false
}

func DisplayNames() []string {
	result := make([]string, 0, len(supportedLanguages))
	for _, language := range supportedLanguages {
		result = append(result, DisplayName(language))
	}
	return result
}

type Translator struct {
	language Language
}

func New(language Language) Translator {
	return Translator{language: Normalize(language)}
}

func (translator Translator) Language() Language {
	return translator.language
}

// Text resolves a message key and optionally evaluates its Go template data.
// English is the fallback for a missing translation; a missing English key is
// returned verbatim to make omissions obvious during development.
func (translator Translator) Text(key string, data ...any) string {
	message, ok := catalogs[translator.language][key]
	if !ok {
		message = catalogs[English][key]
	}
	if message == "" {
		return key
	}
	if len(data) == 0 {
		return message
	}
	parsed, err := template.New(key).Option("missingkey=error").Parse(message)
	if err != nil {
		return message
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data[0]); err != nil {
		return message
	}
	return output.String()
}

// LocalizedError identifies a message in the translation catalog while
// retaining an optional cause for errors.Is and diagnostics.
type LocalizedError struct {
	Key   string
	Data  map[string]any
	cause error
}

func NewError(key string, data map[string]any) error {
	return &LocalizedError{Key: key, Data: cloneData(data)}
}

func WrapError(key string, cause error, data map[string]any) error {
	return &LocalizedError{Key: key, Data: cloneData(data), cause: cause}
}

func (err *LocalizedError) Error() string {
	return New(English).Text(err.Key, err.templateData())
}

func (err *LocalizedError) Unwrap() error {
	return err.cause
}

func (err *LocalizedError) templateData() map[string]any {
	data := cloneData(err.Data)
	if err.cause != nil {
		if _, exists := data["Error"]; !exists {
			data["Error"] = err.cause
		}
	}
	return data
}

// ErrorText returns a user-facing message in the selected language. Errors
// that have not yet been assigned a translation key remain diagnosable but are
// presented with a localized generic prefix.
func (translator Translator) ErrorText(err error) string {
	if err == nil {
		return ""
	}
	var localized *LocalizedError
	if errors.As(err, &localized) {
		data := cloneData(localized.Data)
		if localized.cause != nil {
			if _, exists := data["Error"]; !exists {
				data["Error"] = translator.ErrorText(localized.cause)
			}
		}
		return translator.Text(localized.Key, data)
	}
	return translator.Text("error.unexpected", map[string]any{"Error": err})
}

func ValidateCatalogs() error {
	english := catalogs[English]
	if len(english) == 0 {
		return errors.New("English translation catalog is empty")
	}
	for _, language := range supportedLanguages {
		catalog := catalogs[language]
		for key := range english {
			if catalog[key] == "" {
				return fmt.Errorf("%s translation is missing %q", language, key)
			}
		}
		for key := range catalog {
			if english[key] == "" {
				return fmt.Errorf("%s translation has unknown key %q", language, key)
			}
		}
	}
	return nil
}

func cloneData(data map[string]any) map[string]any {
	if len(data) == 0 {
		return make(map[string]any)
	}
	clone := make(map[string]any, len(data))
	for key, value := range data {
		clone[key] = value
	}
	return clone
}
