package lesson

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for lessons
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new lesson repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetAllLessons returns all lessons
func (r *Repository) GetAllLessons(ctx context.Context) ([]Lesson, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, description, created_at
		FROM lessons
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []Lesson
	for rows.Next() {
		var l Lesson
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan lesson: %w", err)
		}
		lessons = append(lessons, l)
	}
	return lessons, nil
}

// GetLessonByID returns a single lesson by ID
func (r *Repository) GetLessonByID(ctx context.Context, id string) (*Lesson, error) {
	var l Lesson
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, description, created_at
		FROM lessons
		WHERE id = $1
	`, id).Scan(&l.ID, &l.Title, &l.Description, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query lesson by id: %w", err)
	}
	return &l, nil
}

// GetStepsByLessonID returns all steps for a lesson with AR actions joined
func (r *Repository) GetStepsByLessonID(ctx context.Context, lessonID string) ([]LessonStep, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			ls.id, ls.lesson_id, ls.step_id, ls.step_name, ls.content,
			ls.mic_enabled_after, ls.step_order, ls.ar_action_id,
			ar.id, ar.type, ar.target, ar.params
		FROM lesson_steps ls
		LEFT JOIN ar_actions ar ON ls.ar_action_id = ar.id
		WHERE ls.lesson_id = $1
		ORDER BY ls.step_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("query steps: %w", err)
	}
	defer rows.Close()

	var steps []LessonStep
	for rows.Next() {
		var s LessonStep
		var arID *int
		var arType, arTarget *string
		var arParams []byte

		if err := rows.Scan(
			&s.ID, &s.LessonID, &s.StepID, &s.StepName, &s.Content,
			&s.MicEnabledAfter, &s.StepOrder, &s.ARActionID,
			&arID, &arType, &arTarget, &arParams,
		); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}

		// Populate ARAction if exists
		if arID != nil {
			s.ARAction = &ARAction{
				ID:     *arID,
				Type:   *arType,
				Params: arParams,
			}
			if arTarget != nil {
				s.ARAction.Target = *arTarget
			}
		}

		steps = append(steps, s)
	}
	return steps, nil
}

// GetStepByID returns a specific step by lesson_id and step_id with AR action joined
func (r *Repository) GetStepByID(ctx context.Context, lessonID, stepID string) (*LessonStep, error) {
	var s LessonStep
	var arID *int
	var arType, arTarget *string
	var arParams []byte

	err := r.pool.QueryRow(ctx, `
		SELECT
			ls.id, ls.lesson_id, ls.step_id, ls.step_name, ls.content,
			ls.mic_enabled_after, ls.step_order, ls.ar_action_id,
			ar.id, ar.type, ar.target, ar.params
		FROM lesson_steps ls
		LEFT JOIN ar_actions ar ON ls.ar_action_id = ar.id
		WHERE ls.lesson_id = $1 AND ls.step_id = $2
	`, lessonID, stepID).Scan(
		&s.ID, &s.LessonID, &s.StepID, &s.StepName, &s.Content,
		&s.MicEnabledAfter, &s.StepOrder, &s.ARActionID,
		&arID, &arType, &arTarget, &arParams,
	)
	if err != nil {
		return nil, fmt.Errorf("query step by id: %w", err)
	}

	// Populate ARAction if exists
	if arID != nil {
		s.ARAction = &ARAction{
			ID:     *arID,
			Type:   *arType,
			Params: arParams,
		}
		if arTarget != nil {
			s.ARAction.Target = *arTarget
		}
	}

	return &s, nil
}

// GetLessonWithSteps returns a lesson with all its steps
func (r *Repository) GetLessonWithSteps(ctx context.Context, id string) (*LessonWithSteps, error) {
	lesson, err := r.GetLessonByID(ctx, id)
	if err != nil {
		return nil, err
	}

	steps, err := r.GetStepsByLessonID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &LessonWithSteps{
		Lesson: *lesson,
		Steps:  steps,
	}, nil
}

// GetAllARActions returns all AR actions
func (r *Repository) GetAllARActions(ctx context.Context) ([]ARAction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, target, params
		FROM ar_actions
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query ar_actions: %w", err)
	}
	defer rows.Close()

	var actions []ARAction
	for rows.Next() {
		var a ARAction
		var target *string
		if err := rows.Scan(&a.ID, &a.Type, &target, &a.Params); err != nil {
			return nil, fmt.Errorf("scan ar_action: %w", err)
		}
		if target != nil {
			a.Target = *target
		}
		actions = append(actions, a)
	}
	return actions, nil
}
