package api

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"tale-backend/internal/lesson"
	"tale-backend/internal/realtime"
)

var realtimeClient *realtime.Client

// SetRealtimeClient sets the realtime client for audio streaming
func SetRealtimeClient(client *realtime.Client) {
	realtimeClient = client
}

// lessonSessions stores active lesson sessions (in-memory for now)
var (
	lessonSessions = make(map[string]*lesson.Session) // key: lessonID
	sessionMu      sync.RWMutex
)

// RegisterLessonAPI registers REST endpoints for lesson management
func RegisterLessonAPI(repo *lesson.Repository) {
	// GET /lessons - list all lessons
	http.HandleFunc("/lessons", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		lessons, err := repo.GetAllLessons(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, lessons)
	})

	http.HandleFunc("/lessons/", func(w http.ResponseWriter, r *http.Request) {
		// Parse path: /lessons/{id}/...
		path := strings.TrimPrefix(r.URL.Path, "/lessons/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "lesson id required", http.StatusBadRequest)
			return
		}

		lessonID := parts[0]

		// Route based on path and method
		switch {
		// POST /lessons/:id/start
		case len(parts) == 2 && parts[1] == "start" && r.Method == http.MethodPost:
			handleLessonStart(w, r, repo, lessonID)

		// POST /lessons/:id/next
		case len(parts) == 2 && parts[1] == "next" && r.Method == http.MethodPost:
			handleLessonNext(w, r, lessonID)

		// GET /lessons/:id/current
		case len(parts) == 2 && parts[1] == "current" && r.Method == http.MethodGet:
			handleLessonCurrent(w, r, lessonID)

		// GET /lessons/:id
		case len(parts) == 1 && r.Method == http.MethodGet:
			lessonData, err := repo.GetLessonWithSteps(r.Context(), lessonID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, lessonData)

		// GET /lessons/:id/steps
		case len(parts) == 2 && parts[1] == "steps" && r.Method == http.MethodGet:
			steps, err := repo.GetStepsByLessonID(r.Context(), lessonID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, steps)

		// GET /lessons/:id/steps/:stepId
		case len(parts) == 3 && parts[1] == "steps" && r.Method == http.MethodGet:
			stepID := parts[2]
			step, err := repo.GetStepByID(r.Context(), lessonID, stepID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, step)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
}

// handleLessonStart starts a new lesson session
func handleLessonStart(w http.ResponseWriter, r *http.Request, repo *lesson.Repository, lessonID string) {
	log.Printf("[handleLessonStart] called for lessonID=%s", lessonID)
	// Get all steps for this lesson
	steps, err := repo.GetStepsByLessonID(r.Context(), lessonID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	session, err := lesson.NewSession(lessonID, steps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create session
	sessionMu.Lock()
	lessonSessions[lessonID] = session
	sessionMu.Unlock()

	// Send audio for first step
	speakLessonStep(session.Current())

	// Return first step
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "started",
		"lesson_id":   lessonID,
		"total_steps": session.TotalSteps(),
		"current":     0,
		"step":        session.Current(),
	})
}

// handleLessonNext advances to the next step
func handleLessonNext(w http.ResponseWriter, r *http.Request, lessonID string) {
	sessionMu.Lock()
	session, exists := lessonSessions[lessonID]
	if !exists {
		sessionMu.Unlock()
		http.Error(w, "lesson not started", http.StatusBadRequest)
		return
	}

	currentStep, completed := session.Advance()

	if completed {
		delete(lessonSessions, lessonID)
		sessionMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "completed",
			"lesson_id": lessonID,
		})
		return
	}

	totalSteps := session.TotalSteps()
	currentIdx := session.CurrentIndex()
	sessionMu.Unlock()

	// Send audio for current step
	speakLessonStep(currentStep)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "in_progress",
		"lesson_id":   lessonID,
		"total_steps": totalSteps,
		"current":     currentIdx,
		"step":        currentStep,
	})
}

// speakLessonStep sends TTS audio for a lesson step
func speakLessonStep(step lesson.LessonStep) {
	if realtimeClient == nil {
		log.Println("[Lesson] No realtime client, skipping audio")
		return
	}

	var arAction *realtime.LessonARAction
	if step.ARAction != nil {
		arAction = &realtime.LessonARAction{
			Type:   step.ARAction.Type,
			Target: step.ARAction.Target,
			Params: step.ARAction.Params,
		}
	}

	go func() {
		if err := realtimeClient.SpeakLessonStep(step.StepID, step.StepName, step.Content, arAction); err != nil {
			log.Printf("[Lesson] TTS error: %v", err)
		}
	}()
}

// handleLessonCurrent returns the current step
func handleLessonCurrent(w http.ResponseWriter, r *http.Request, lessonID string) {
	sessionMu.RLock()
	session, exists := lessonSessions[lessonID]
	if !exists {
		sessionMu.RUnlock()
		http.Error(w, "lesson not started", http.StatusBadRequest)
		return
	}

	currentStep := session.Current()
	totalSteps := session.TotalSteps()
	current := session.CurrentIndex()
	sessionMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "in_progress",
		"lesson_id":   lessonID,
		"total_steps": totalSteps,
		"current":     current,
		"step":        currentStep,
	})
}
