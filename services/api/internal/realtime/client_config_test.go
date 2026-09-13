package realtime

import (
	"net/url"
	"testing"
)

func TestRealtimeAPIURLUsesConfiguredModel(t *testing.T) {
	t.Setenv("OPENAI_REALTIME_MODEL", "realtime/custom model")

	parsed, err := url.Parse(realtimeAPIURL())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "wss" || parsed.Host != "api.openai.com" || parsed.Path != "/v1/realtime" {
		t.Fatalf("unexpected realtime URL: %s", parsed)
	}
	if got := parsed.Query().Get("model"); got != "realtime/custom model" {
		t.Fatalf("model = %q", got)
	}
}

func TestRealtimeAPIURLUsesDefaultModel(t *testing.T) {
	t.Setenv("OPENAI_REALTIME_MODEL", "")

	parsed, err := url.Parse(realtimeAPIURL())
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("model"); got != DefaultRealtimeModel {
		t.Fatalf("model = %q, want %q", got, DefaultRealtimeModel)
	}
}
