-- Migration 000019 — AI conversation memory (Mission Control Phase B.2 / AI-003).
--
-- Adds persistent conversation threads for the Mission Control dock. A
-- conversation is a server-side grouping of related submissions: successive
-- prompts from the same user within a 60-minute window append to the same
-- conversation; outside that window a new one is created. The lifecycle is
-- server-decided (see services/orchestrator/internal/handlers/conversations.go
-- ensureActiveConversation) and requires no UI affordance in B.2.
--
-- Tables are additive only. The new ai_tasks.conversation_id column is
-- nullable so pre-B.2 task rows continue to satisfy the FK with NULL.

CREATE TABLE ai_conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT,
    state       VARCHAR(31) NOT NULL DEFAULT 'active',
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_conversations_state_check CHECK (state IN ('active', 'archived'))
);

CREATE INDEX idx_ai_conversations_user_updated
    ON ai_conversations(created_by, updated_at DESC);
CREATE INDEX idx_ai_conversations_org_updated
    ON ai_conversations(org_id, updated_at DESC);

CREATE TABLE ai_conversation_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(15) NOT NULL,
    content         TEXT NOT NULL,
    task_id         UUID REFERENCES ai_tasks(id) ON DELETE SET NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_conversation_messages_role_check
        CHECK (role IN ('user', 'assistant', 'system'))
);

CREATE INDEX idx_ai_conversation_messages_conv_created
    ON ai_conversation_messages(conversation_id, created_at);

ALTER TABLE ai_tasks
    ADD COLUMN conversation_id UUID REFERENCES ai_conversations(id) ON DELETE SET NULL;

CREATE INDEX idx_ai_tasks_conversation
    ON ai_tasks(conversation_id) WHERE conversation_id IS NOT NULL;
