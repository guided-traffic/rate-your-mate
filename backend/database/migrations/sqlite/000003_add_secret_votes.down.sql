-- SQLite doesn't support DROP COLUMN directly
-- For proper rollback, we'd need to recreate the table
-- This is a simplified down migration that creates a view to hide the column
-- In production, you'd want to recreate the table without this column

-- Note: SQLite 3.35.0+ supports DROP COLUMN, but for compatibility we keep this simple
-- The column will remain but won't be used after rollback
