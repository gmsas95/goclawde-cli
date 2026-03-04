-- Migration: Add conversation_summaries table
-- Purpose: Cache conversation summaries for long conversations to improve performance
-- Created: 2026-03-04

-- Create conversation_summaries table
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id TEXT PRIMARY KEY,
    conv_id TEXT NOT NULL,
    summary TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient conversation lookups
CREATE INDEX IF NOT EXISTS idx_conv_summaries_conv_id ON conversation_summaries(conv_id);

-- Create index for cleanup of old summaries
CREATE INDEX IF NOT EXISTS idx_conv_summaries_created_at ON conversation_summaries(created_at);

-- Add comment for documentation
COMMENT ON TABLE conversation_summaries IS 'Cached summaries of long conversations for context window management';
COMMENT ON COLUMN conversation_summaries.conv_id IS 'Reference to the conversation being summarized';
COMMENT ON COLUMN conversation_summaries.summary IS 'The generated summary text';
