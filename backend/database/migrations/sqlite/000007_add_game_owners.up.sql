-- Add game_owners table to track which users own which games (SQLite)

CREATE TABLE IF NOT EXISTS game_owners (
    app_id INTEGER NOT NULL,
    steam_id TEXT NOT NULL,
    playtime_forever INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (app_id, steam_id)
);

-- Index for looking up all owners of a game
CREATE INDEX IF NOT EXISTS idx_game_owners_app_id ON game_owners(app_id);

-- Index for looking up all games a user owns
CREATE INDEX IF NOT EXISTS idx_game_owners_steam_id ON game_owners(steam_id);
