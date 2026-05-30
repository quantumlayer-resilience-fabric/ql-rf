-- Reverse migration 000019.

DROP INDEX IF EXISTS idx_ai_tasks_conversation;
ALTER TABLE ai_tasks DROP COLUMN IF EXISTS conversation_id;

DROP TABLE IF EXISTS ai_conversation_messages;
DROP TABLE IF EXISTS ai_conversations;
