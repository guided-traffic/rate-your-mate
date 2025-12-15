-- Add is_invalidated column to votes table
ALTER TABLE votes ADD COLUMN is_invalidated INTEGER DEFAULT 0;
