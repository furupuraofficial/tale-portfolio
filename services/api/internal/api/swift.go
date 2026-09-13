package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tale-backend/internal/ar"
	"tale-backend/internal/realtime"
)

// RegisterSwiftAPI registers REST endpoints that a Swift/iOS client can call.
// All handlers are added to the default http mux used by ARServer.
func RegisterSwiftAPI(arServer *ar.Server, rc *realtime.Client) {
	_ = arServer // currently not needed, reserved for future bidirectional use
	// Current state (step, mode, user)
	http.HandleFunc("/conversation/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"step":     rc.GetCurrentStep(),
			"mode":     rc.GetMode(),
			"userName": rc.GetUserName(),
		})
	})

	// Conversation history (optional ?limit=50)
	http.HandleFunc("/conversation/history", func(w http.ResponseWriter, r *http.Request) {
		limit := parseLimit(r, 100)
		writeJSON(w, http.StatusOK, rc.GetHistory(limit))
	})

	// Rules list (for UI/diagnostic)
	http.HandleFunc("/conversation/rules", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, realtime.RuleDatabase)
	})

	// Live transcript of the current sentence being spoken (SSE)
	http.HandleFunc("/conversation/transcript/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := rc.SubscribeTranscript()
		defer rc.UnsubscribeTranscript(ch)

		ctxDone := r.Context().Done()
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				writeSSE(w, ev)
				flusher.Flush()
			case <-ctxDone:
				return
			}
		}
	})

	// Live audio stream (SSE): streams base64 PCM chunks with metadata
	http.HandleFunc("/conversation/audio/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := rc.SubscribeAudioStream()
		defer rc.UnsubscribeAudioStream(ch)

		ctxDone := r.Context().Done()
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				writeSSE(w, ev)
				flusher.Flush()
			case <-ctxDone:
				return
			}
		}
	})

	// Update stored user name
	http.HandleFunc("/conversation/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rc.SetUserName(body.Name)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Language preference: GET to read, POST to set
	http.HandleFunc("/conversation/language", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]string{
				"language": rc.GetPreferredLanguage(),
			})
		case http.MethodPost:
			var body struct {
				Language string `json:"language"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := rc.UpdateLanguagePreference(body.Language); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"status":   "ok",
				"language": rc.GetPreferredLanguage(),
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Start scripted conversation (frontend-triggered)
	http.HandleFunc("/conversation/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Start audio + scripted conversation
		rc.SetMode(realtime.ModeScript)
		rc.SetCurrentStep(realtime.StepIntro)
		if err := rc.StartAudioStream(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = rc.StartScriptedConversation()
		writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
	})

	// Pause / resume conversation (suppress realtime + stop mic send)
	http.HandleFunc("/conversation/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Default to pause=true if no body is provided (for Swift client compatibility)
		pause := true
		var body struct {
			Pause *bool `json:"pause"` // Use pointer to detect if field was provided
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Pause != nil {
				pause = *body.Pause
			}
		}

		if pause {
			rc.PauseRealtime()
			writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
			return
		}

		rc.ResumeRealtime()
		writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
	})

	// Explicit resume endpoint (alias for pause=false)
	http.HandleFunc("/conversation/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		rc.ResumeRealtime()
		writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
	})
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeSSE sends a Server-Sent Event with JSON payload.
func writeSSE(w http.ResponseWriter, v interface{}) {
	data, _ := json.Marshal(v)
	w.Write([]byte("data: "))
	w.Write(data)
	w.Write([]byte("\n\n"))
}

func parseLimit(r *http.Request, def int) int {
	q := r.URL.Query().Get("limit")
	if q == "" {
		return def
	}
	n, err := strconv.Atoi(q)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
