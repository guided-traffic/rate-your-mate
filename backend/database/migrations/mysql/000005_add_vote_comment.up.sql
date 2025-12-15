-- Add comment column to votes table
ALTER TABLE votes ADD COLUMN comment VARCHAR(160) DEFAULT NULL;
