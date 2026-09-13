package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"tale-backend/internal/realtime"
)

const maxRuleBodyBytes = 1 << 20

// RegisterRuleAPI registers the CRUD endpoints used by the admin web app.
func RegisterRuleAPI() {
	http.HandleFunc("/rule/items", handleRuleCollection)
	http.HandleFunc("/rule/items/", handleRuleItem)
}

func handleRuleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, realtime.ListRules())
	case http.MethodPost:
		var rule realtime.Rule
		if err := decodeRule(w, r, &rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		created, err := realtime.CreateRule(rule)
		if err != nil {
			writeRuleError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func handleRuleItem(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimPrefix(r.URL.Path, "/rule/items/")
	id, err := url.PathUnescape(rawID)
	if err != nil || id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid rule id is required"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var rule realtime.Rule
		if err := decodeRule(w, r, &rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		updated, err := realtime.UpdateRule(id, rule)
		if err != nil {
			writeRuleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := realtime.DeleteRule(id); err != nil {
			writeRuleError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "PUT, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func decodeRule(w http.ResponseWriter, r *http.Request, destination *realtime.Rule) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRuleBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func writeRuleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, realtime.ErrRuleNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, realtime.ErrRuleExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}
