-- Add is_secret column to votes table
-- Default is 0 (false/open) for existing votes
ALTER TABLE votes ADD COLUMN is_secret INTEGER DEFAULT 0;
