-- Remove game_owners table (SQLite)

DROP INDEX IF EXISTS idx_game_owners_steam_id;
DROP INDEX IF EXISTS idx_game_owners_app_id;
DROP TABLE IF EXISTS game_owners;
