-- Enable the pgvector extension inside the brainkit database.
-- Runs once on first container boot, before the example connects.
CREATE EXTENSION IF NOT EXISTS vector;
