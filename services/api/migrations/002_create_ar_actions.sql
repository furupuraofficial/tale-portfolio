-- AR Actions master table
CREATE TABLE IF NOT EXISTS ar_actions (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    target VARCHAR(100),
    params JSONB DEFAULT '{}'
);

-- Index for searching by type
CREATE INDEX IF NOT EXISTS idx_ar_actions_type ON ar_actions(type);

-- Add ar_action_id foreign key to lesson_steps (replaces ar_actions JSONB)
ALTER TABLE lesson_steps
    DROP COLUMN IF EXISTS ar_actions,
    ADD COLUMN IF NOT EXISTS ar_action_id INT REFERENCES ar_actions(id) ON DELETE SET NULL;

-- Index for the foreign key
CREATE INDEX IF NOT EXISTS idx_lesson_steps_ar_action ON lesson_steps(ar_action_id);
