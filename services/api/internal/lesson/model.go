package lesson

import (
	"encoding/json"
	"time"
)

// Lesson represents a mini Japanese lesson
type Lesson struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// ARAction represents an AR command to be executed on the frontend
type ARAction struct {
	ID     int             `json:"id"`
	Type   string          `json:"type"`
	Target string          `json:"target,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// LessonStep represents a single step in a lesson
type LessonStep struct {
	ID              int       `json:"id"`
	LessonID        string    `json:"lesson_id"`
	StepID          string    `json:"step_id"`
	StepName        string    `json:"step_name"`
	Content         string    `json:"content"`
	MicEnabledAfter bool      `json:"mic_enabled_after"`
	StepOrder       int       `json:"step_order"`
	ARActionID      *int      `json:"ar_action_id,omitempty"`
	ARAction        *ARAction `json:"ar_action,omitempty"`
}

// LessonWithSteps combines a lesson with its steps
type LessonWithSteps struct {
	Lesson
	Steps []LessonStep `json:"steps"`
}
