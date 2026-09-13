package realtime

import "testing"

func TestGeminiModelName(t *testing.T) {
	t.Setenv("GEMINI_MODEL", "")
	if got := geminiModelName(); got != DefaultGeminiModel {
		t.Fatalf("geminiModelName() = %q, want %q", got, DefaultGeminiModel)
	}

	t.Setenv("GEMINI_MODEL", " gemini-custom ")
	if got := geminiModelName(); got != "gemini-custom" {
		t.Fatalf("geminiModelName() = %q, want gemini-custom", got)
	}
}
