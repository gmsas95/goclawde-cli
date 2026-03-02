-- Migration: Create v2 schema for content block architecture
-- This migration creates new tables for the modernized conversation storage

-- Create conversations_v2 table
CREATE TABLE IF NOT EXISTS conversations_v2 (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for user lookups
CREATE INDEX idx_conversations_v2_user_id ON conversations_v2(user_id);

-- Create messages_v2 table with content blocks
CREATE TABLE IF NOT EXISTS messages_v2 (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations_v2(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content JSONB NOT NULL DEFAULT '[]'::jsonb,  -- Array of content blocks
    metadata JSONB DEFAULT '{}'::jsonb,          -- Usage, stop_reason, error_message, etc.
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX idx_messages_v2_conversation_id ON messages_v2(conversation_id);
CREATE INDEX idx_messages_v2_created_at ON messages_v2(created_at);
CREATE INDEX idx_messages_v2_conversation_created ON messages_v2(conversation_id, created_at);

-- Migration function: Migrate old messages to new format
-- This can be run manually after deployment
CREATE OR REPLACE FUNCTION migrate_conversation_to_v2(old_conv_id TEXT)
RETURNS TEXT AS $$
DECLARE
    new_conv_id TEXT;
    msg_record RECORD;
    content_blocks JSONB;
    metadata JSONB;
BEGIN
    -- Generate new conversation ID
    new_conv_id := gen_random_uuid()::TEXT;
    
    -- Get user_id from old conversation
    INSERT INTO conversations_v2 (id, user_id, created_at, updated_at)
    SELECT new_conv_id, user_id, created_at, updated_at
    FROM conversations
    WHERE id = old_conv_id;
    
    -- Migrate messages
    FOR msg_record IN 
        SELECT * FROM messages 
        WHERE conversation_id = old_conv_id 
        ORDER BY created_at ASC
    LOOP
        -- Build content blocks array
        content_blocks := '[]'::jsonb;
        
        -- Add text content if present
        IF msg_record.content IS NOT NULL AND length(msg_record.content) > 0 THEN
            content_blocks := content_blocks || jsonb_build_object(
                'type', 'text',
                'text', msg_record.content
            );
        END IF;
        
        -- Add tool calls if present
        IF msg_record.tool_calls IS NOT NULL AND msg_record.tool_calls != '[]'::jsonb THEN
            FOR i IN 0..jsonb_array_length(msg_record.tool_calls) - 1 LOOP
                content_blocks := content_blocks || jsonb_build_object(
                    'type', 'tool_call',
                    'id', msg_record.tool_calls->i->>'id',
                    'name', msg_record.tool_calls->i->>'name',
                    'arguments', msg_record.tool_calls->i->'arguments'
                );
            END LOOP;
        END IF;
        
        -- Build metadata
        metadata := jsonb_build_object(
            'input_tokens', msg_record.tokens,
            'output_tokens', 0
        );
        
        -- Insert into new table
        INSERT INTO messages_v2 (id, conversation_id, role, content, metadata, created_at)
        VALUES (
            gen_random_uuid()::TEXT,
            new_conv_id,
            msg_record.role,
            content_blocks,
            metadata,
            msg_record.created_at
        );
    END LOOP;
    
    RETURN new_conv_id;
END;
$$ LANGUAGE plpgsql;

-- Note: Tool results from old format need special handling
-- Old format stored tool results as regular messages with role='tool'
-- New format uses ToolResultBlock within content array