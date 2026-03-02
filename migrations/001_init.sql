-- Migration: Create v2 schema for content block architecture
-- PostgreSQL version with JSONB support

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create conversations_v2 table
CREATE TABLE IF NOT EXISTS conversations_v2 (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for user lookups
CREATE INDEX IF NOT EXISTS idx_conversations_v2_user_id ON conversations_v2(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_v2_updated_at ON conversations_v2(updated_at);

-- Create messages_v2 table with content blocks using JSONB
CREATE TABLE IF NOT EXISTS messages_v2 (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations_v2(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content JSONB NOT NULL DEFAULT '[]'::jsonb,  -- Array of content blocks
    metadata JSONB DEFAULT '{}'::jsonb,          -- Usage, stop_reason, error, etc.
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_messages_v2_conversation_id ON messages_v2(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_v2_created_at ON messages_v2(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_v2_conversation_created ON messages_v2(conversation_id, created_at DESC);

-- GIN index for JSONB queries (optional, for advanced searching)
CREATE INDEX IF NOT EXISTS idx_messages_v2_content_gin ON messages_v2 USING GIN (content);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
DROP TRIGGER IF EXISTS update_conversations_v2_updated_at ON conversations_v2;
CREATE TRIGGER update_conversations_v2_updated_at
    BEFORE UPDATE ON conversations_v2
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();