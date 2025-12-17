-- Add game_owners table to track which users own which games (MySQL)

CREATE TABLE IF NOT EXISTS game_owners (
    app_id BIGINT UNSIGNED NOT NULL,
    steam_id VARCHAR(20) NOT NULL,
    playtime_forever INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (app_id, steam_id),
    INDEX idx_game_owners_app_id (app_id),
    INDEX idx_game_owners_steam_id (steam_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
