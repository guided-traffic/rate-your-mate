-- Extend steam_id columns from VARCHAR(20) to VARCHAR(50) to support FAKE_ prefixed IDs

ALTER TABLE users MODIFY COLUMN steam_id VARCHAR(50) NOT NULL;
ALTER TABLE banned_users MODIFY COLUMN steam_id VARCHAR(50) NOT NULL;
ALTER TABLE banned_users MODIFY COLUMN banned_by VARCHAR(50) NOT NULL;
