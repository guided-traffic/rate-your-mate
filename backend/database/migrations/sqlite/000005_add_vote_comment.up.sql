-- Add comment column to votes table
ALTER TABLE votes ADD COLUMN comment TEXT DEFAULT NULL;
