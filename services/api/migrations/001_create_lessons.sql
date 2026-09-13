-- Lessons table
CREATE TABLE IF NOT EXISTS lessons (
    id VARCHAR(10) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Lesson steps table
CREATE TABLE IF NOT EXISTS lesson_steps (
    id SERIAL PRIMARY KEY,
    lesson_id VARCHAR(10) NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    step_id VARCHAR(50) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    mic_enabled_after BOOLEAN DEFAULT false,
    ar_actions JSONB DEFAULT '[]',
    step_order INT NOT NULL,
    UNIQUE(lesson_id, step_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_lesson_steps_lesson_id ON lesson_steps(lesson_id);
CREATE INDEX IF NOT EXISTS idx_lesson_steps_order ON lesson_steps(lesson_id, step_order);
