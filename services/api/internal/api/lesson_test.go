package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestWriteLessonRepositoryError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing row",
			err:        errors.Join(errors.New("query lesson"), pgx.ErrNoRows),
			wantStatus: http.StatusNotFound,
			wantBody:   "lesson resource not found",
		},
		{
			name:       "database failure",
			err:        errors.New("connection includes internal details"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeLessonRepositoryError(response, "test operation", test.err)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", response.Body.String(), test.wantBody)
			}
			if strings.Contains(response.Body.String(), "internal details") {
				t.Fatalf("response leaked internal error: %q", response.Body.String())
			}
		})
	}
}
