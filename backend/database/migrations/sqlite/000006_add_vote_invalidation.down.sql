-- SQLite does not support DROP COLUMN directly in older versions
-- Create a new table without the column, copy data, and rename
CREATE TABLE votes_temp (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_user_id INTEGER NOT NULL REFERENCES users(id),
    to_user_id INTEGER NOT NULL REFERENCES users(id),
    achievement_id TEXT NOT NULL,
    points INTEGER DEFAULT 1,
    is_secret INTEGER DEFAULT 0,
    comment TEXT DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (from_user_id != to_user_id)
);

INSERT INTO votes_temp (id, from_user_id, to_user_id, achievement_id, points, is_secret, comment, created_at)
SELECT id, from_user_id, to_user_id, achievement_id, points, is_secret, comment, created_at FROM votes;

DROP TABLE votes;

ALTER TABLE votes_temp RENAME TO votes;

CREATE INDEX idx_votes_achievement ON votes(achievement_id, to_user_id);
CREATE INDEX idx_votes_timeline ON votes(created_at DESC);
