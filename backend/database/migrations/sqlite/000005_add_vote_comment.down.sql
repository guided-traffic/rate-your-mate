-- SQLite does not support DROP COLUMN directly, so we need to recreate the table
-- This is a simplified version - in production you'd need to copy data

-- Create new table without comment column
CREATE TABLE votes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id TEXT NOT NULL,
    points INTEGER DEFAULT 1,
    is_secret INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (from_user_id != to_user_id)
);

-- Copy data
INSERT INTO votes_new (id, from_user_id, to_user_id, achievement_id, points, is_secret, created_at)
SELECT id, from_user_id, to_user_id, achievement_id, points, is_secret, created_at FROM votes;

-- Drop old table
DROP TABLE votes;

-- Rename new table
ALTER TABLE votes_new RENAME TO votes;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_votes_achievement ON votes(achievement_id, to_user_id);
CREATE INDEX IF NOT EXISTS idx_votes_timeline ON votes(created_at DESC);
