package ar

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginAllowed(t *testing.T) {
	t.Setenv("TALE_ALLOWED_ORIGINS", "https://admin.example.com, http://localhost:3000")

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "native client", want: true},
		{name: "same origin", origin: "https://api.example.com", want: true},
		{name: "configured origin", origin: "https://admin.example.com", want: true},
		{name: "configured localhost", origin: "http://localhost:3000", want: true},
		{name: "untrusted origin", origin: "https://attacker.example", want: false},
		{name: "origin with path", origin: "https://admin.example.com/path", want: false},
		{name: "non-http origin", origin: "file://local", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://api.example.com/ws", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := websocketOriginAllowed(request); got != test.want {
				t.Fatalf("websocketOriginAllowed() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := response.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
}
