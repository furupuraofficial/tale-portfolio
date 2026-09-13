package lesson

import "fmt"

// Session owns the progress of one lesson for one learner.
type Session struct {
	lessonID    string
	steps       []LessonStep
	currentStep int
}

func NewSession(lessonID string, steps []LessonStep) (*Session, error) {
	if lessonID == "" {
		return nil, fmt.Errorf("lesson id is required")
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("lesson has no steps")
	}

	ownedSteps := append([]LessonStep(nil), steps...)
	return &Session{lessonID: lessonID, steps: ownedSteps}, nil
}

func (s *Session) LessonID() string {
	return s.lessonID
}

func (s *Session) TotalSteps() int {
	return len(s.steps)
}

func (s *Session) CurrentIndex() int {
	return s.currentStep
}

func (s *Session) Current() LessonStep {
	return s.steps[s.currentStep]
}

// Advance moves to the next step. The second return value is true when the
// session has completed and no current step remains.
func (s *Session) Advance() (LessonStep, bool) {
	s.currentStep++
	if s.currentStep >= len(s.steps) {
		return LessonStep{}, true
	}
	return s.Current(), false
}
