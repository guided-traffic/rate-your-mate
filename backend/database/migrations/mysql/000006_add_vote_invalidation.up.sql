-- Add is_invalidated column to votes table
ALTER TABLE votes ADD COLUMN is_invalidated TINYINT(1) DEFAULT 0;
