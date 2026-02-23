-- Myrai 2.0 Database Migration
-- Run this to upgrade from Myrai 1.x to 2.0

-- ============================================
-- Phase 1: Neural Clusters
-- ============================================

CREATE TABLE IF NOT EXISTS neural_clusters (
    id TEXT PRIMARY KEY,
    theme TEXT NOT NULL,
    essence TEXT NOT NULL,
    memory_ids JSON NOT NULL,
    embedding VECTOR(1536),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP,
    access_count INTEGER DEFAULT 0,
    confidence_score FLOAT DEFAULT 0.0,
    cluster_size INTEGER DEFAULT 0,
    metadata JSON
);

CREATE TABLE IF NOT EXISTS cluster_formation_logs (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL REFERENCES neural_clusters(id),
    formation_type TEXT NOT NULL,
    memory_count INTEGER,
    formed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    formation_metadata JSON
);

CREATE INDEX IF NOT EXISTS idx_clusters_theme ON neural_clusters(theme);
CREATE INDEX IF NOT EXISTS idx_clusters_created ON neural_clusters(created_at);
CREATE INDEX IF NOT EXISTS idx_clusters_accessed ON neural_clusters(last_accessed);
CREATE INDEX IF NOT EXISTS idx_clusters_embedding ON neural_clusters USING ivfflat (embedding vector_cosine_ops);

-- ============================================
-- Phase 2: Skills & MCP
-- ============================================

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    author TEXT,
    source TEXT NOT NULL,
    source_url TEXT,
    manifest JSON NOT NULL,
    status TEXT DEFAULT 'disabled',
    error_message TEXT,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_loaded_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS mcp_servers (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    command TEXT NOT NULL,
    args JSON,
    env JSON,
    enabled BOOLEAN DEFAULT false,
    status TEXT DEFAULT 'stopped',
    error_message TEXT,
    container_id TEXT,
    started_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_skills_status ON skills(status);
CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source);

-- ============================================
-- Phase 3: Persona Evolution
-- ============================================

CREATE TABLE IF NOT EXISTS persona_versions (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    identity JSON NOT NULL,
    user_profile JSON NOT NULL,
    change_type TEXT NOT NULL,
    change_description TEXT,
    triggered_by TEXT
);

CREATE TABLE IF NOT EXISTS evolution_proposals (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    rationale TEXT,
    change JSON NOT NULL,
    confidence FLOAT NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP,
    response TEXT
);

CREATE INDEX IF NOT EXISTS idx_proposals_status ON evolution_proposals(status);

-- ============================================
-- Phase 4: Tool Chains
-- ============================================

CREATE TABLE IF NOT EXISTS tool_chains (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    version TEXT,
    author TEXT,
    type TEXT NOT NULL,
    definition JSON NOT NULL,
    is_builtin BOOLEAN DEFAULT false,
    is_shared BOOLEAN DEFAULT false,
    install_count INTEGER DEFAULT 0,
    rating FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chain_executions (
    id TEXT PRIMARY KEY,
    chain_id TEXT REFERENCES tool_chains(id),
    status TEXT,
    input_params JSON,
    results JSON,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- ============================================
-- Phase 5: Reflection Engine
-- ============================================

CREATE TABLE IF NOT EXISTS contradictions (
    id TEXT PRIMARY KEY,
    memory_a_id TEXT NOT NULL REFERENCES memories(id),
    memory_b_id TEXT NOT NULL REFERENCES memories(id),
    severity TEXT NOT NULL,
    description TEXT NOT NULL,
    suggested_resolution TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',
    resolved_at TIMESTAMP,
    resolution TEXT
);

CREATE TABLE IF NOT EXISTS redundancy_groups (
    id TEXT PRIMARY KEY,
    theme TEXT,
    memory_ids JSON NOT NULL,
    cluster_ids JSON,
    reason TEXT,
    suggested_action TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',
    consolidated_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS knowledge_gaps (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    mention_count INTEGER,
    memory_count INTEGER,
    gap_ratio FLOAT,
    suggested_action TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',
    filled_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS reflection_reports (
    id TEXT PRIMARY KEY,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    overall_score INTEGER,
    total_memories INTEGER,
    neural_clusters INTEGER,
    contradictions INTEGER,
    redundancies INTEGER,
    gaps INTEGER,
    details JSON,
    suggested_actions JSON
);

CREATE INDEX IF NOT EXISTS idx_contradictions_status ON contradictions(status);
CREATE INDEX IF NOT EXISTS idx_redundancies_status ON redundancy_groups(status);

-- ============================================
-- Phase 6: Marketplace
-- ============================================

CREATE TABLE IF NOT EXISTS marketplace_agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT,
    repository_url TEXT,
    manifest JSON NOT NULL,
    download_count INTEGER DEFAULT 0,
    rating FLOAT,
    review_count INTEGER DEFAULT 0,
    is_verified BOOLEAN DEFAULT false,
    is_official BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

CREATE TABLE IF NOT EXISTS installed_agents (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES marketplace_agents(id),
    installed_version TEXT,
    config JSON,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS agent_reviews (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES marketplace_agents(id),
    user_id TEXT,
    rating INTEGER,
    review TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_marketplace_downloads ON marketplace_agents(download_count);
CREATE INDEX IF NOT EXISTS idx_marketplace_rating ON marketplace_agents(rating);

-- ============================================
-- Migration Complete
-- ============================================

INSERT INTO schema_migrations (version, applied_at) VALUES ('2.0.0', CURRENT_TIMESTAMP);
