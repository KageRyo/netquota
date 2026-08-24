package i18n

import (
	"errors"
	"testing"
)

func TestCatalogsAreComplete(t *testing.T) {
	t.Parallel()

	if err := ValidateCatalogs(); err != nil {
		t.Fatalf("ValidateCatalogs: %v", err)
	}
}

func TestLanguageNamesAndFallback(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		language Language
		name     string
	}{
		{English, "English"},
		{TraditionalChinese, "正體中文"},
		{Japanese, "日本語"},
	} {
		if got := DisplayName(test.language); got != test.name {
			t.Fatalf("DisplayName(%q) = %q, want %q", test.language, got, test.name)
		}
		if got, ok := ParseDisplayName(test.name); !ok || got != test.language {
			t.Fatalf("ParseDisplayName(%q) = (%q, %v)", test.name, got, ok)
		}
	}
	if got := Normalize("unsupported"); got != English {
		t.Fatalf("Normalize fallback = %q, want %q", got, English)
	}
}

func TestTextRendersTemplateAndLocalizedErrors(t *testing.T) {
	t.Parallel()

	translator := New(TraditionalChinese)
	if got, want := translator.Text("tray.update_available", map[string]any{"Version": "v0.2.0"}), "有可用更新：v0.2.0"; got != want {
		t.Fatalf("translated update label = %q, want %q", got, want)
	}
	cause := NewError("error.quota_negative", nil)
	err := WrapError("error.settings.total_quota", cause, nil)
	if got, want := translator.ErrorText(err), "總流量上限：流量上限不可為負數。"; got != want {
		t.Fatalf("translated error = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("localized error should retain its cause")
	}
}
