-- Revert steam_id columns back to VARCHAR(20)

ALTER TABLE users MODIFY COLUMN steam_id VARCHAR(20) NOT NULL;
ALTER TABLE banned_users MODIFY COLUMN steam_id VARCHAR(20) NOT NULL;
ALTER TABLE banned_users MODIFY COLUMN banned_by VARCHAR(20) NOT NULL;
