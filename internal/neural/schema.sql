-- Neural Clusters Database Schema
-- Phase 1: Neural Cluster Implementation for Myrai 2.0
-- Supports pgvector extension for vector similarity search

-- ============================================
-- Enable pgvector extension (PostgreSQL only)
-- ============================================
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================
-- Neural Clusters Table
-- ============================================
CREATE TABLE IF NOT EXISTS neural_clusters (
    id VARCHAR(64) PRIMARY KEY,
    theme VARCHAR(255) NOT NULL,
    essence TEXT NOT NULL,
    memory_ids TEXT NOT NULL, -- JSON array of memory IDs
    embedding BLOB, -- Vector embedding for similarity search (VECTOR(1536) for pgvector)
    access_count INTEGER DEFAULT 0,
    confidence_score REAL DEFAULT 0.0,
    cluster_size INTEGER DEFAULT 0,
    metadata TEXT, -- JSON object for extensibility
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP
);

-- ============================================
-- Cluster Formation Logs Table
-- ============================================
CREATE TABLE IF NOT EXISTS cluster_formation_logs (
    id VARCHAR(64) PRIMARY KEY,
    cluster_id VARCHAR(64),
    operation VARCHAR(50) NOT NULL, -- 'create', 'merge', 'split', 'refresh', 'delete'
    details TEXT, -- JSON object with operation details
    memory_count INTEGER DEFAULT 0,
    previous_state TEXT, -- JSON of previous state (for audit)
    new_state TEXT, -- JSON of new state (for audit)
    duration_ms INTEGER, -- Operation duration in milliseconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Cluster-Memory Association Table (for efficient queries)
-- ============================================
CREATE TABLE IF NOT EXISTS cluster_memories (
    cluster_id VARCHAR(64) NOT NULL,
    memory_id VARCHAR(64) NOT NULL,
    similarity_score REAL DEFAULT 0.0,
    is_representative BOOLEAN DEFAULT FALSE, -- Whether this memory is the cluster representative
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (cluster_id, memory_id),
    FOREIGN KEY (cluster_id) REFERENCES neural_clusters(id) ON DELETE CASCADE
);

-- ============================================
-- Query Patterns Table (for learning cluster usage)
-- ============================================
CREATE TABLE IF NOT EXISTS cluster_query_patterns (
    id VARCHAR(64) PRIMARY KEY,
    query_text TEXT NOT NULL,
    query_embedding BLOB, -- Vector embedding
    matched_cluster_ids TEXT, -- JSON array of cluster IDs
    matched_memory_ids TEXT, -- JSON array of memory IDs
    tokens_used INTEGER DEFAULT 0,
    latency_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Indexes for Performance
-- ============================================

-- Neural clusters indexes
CREATE INDEX IF NOT EXISTS idx_clusters_theme ON neural_clusters(theme);
CREATE INDEX IF NOT EXISTS idx_clusters_confidence ON neural_clusters(confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_clusters_size ON neural_clusters(cluster_size DESC);
CREATE INDEX IF NOT EXISTS idx_clusters_access ON neural_clusters(access_count DESC);
CREATE INDEX IF NOT EXISTS idx_clusters_created ON neural_clusters(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_clusters_updated ON neural_clusters(updated_at DESC);

-- Cluster formation logs indexes
CREATE INDEX IF NOT EXISTS idx_formation_cluster ON cluster_formation_logs(cluster_id);
CREATE INDEX IF NOT EXISTS idx_formation_operation ON cluster_formation_logs(operation);
CREATE INDEX IF NOT EXISTS idx_formation_created ON cluster_formation_logs(created_at DESC);

-- Cluster memories indexes
CREATE INDEX IF NOT EXISTS idx_cm_memory ON cluster_memories(memory_id);
CREATE INDEX IF NOT EXISTS idx_cm_similarity ON cluster_memories(similarity_score DESC);

-- Query patterns indexes
CREATE INDEX IF NOT EXISTS idx_patterns_created ON cluster_query_patterns(created_at DESC);

-- ============================================
-- Vector Search Index (PostgreSQL with pgvector)
-- ============================================
-- Note: This requires pgvector extension
-- For SQLite/Badger: Vector search is performed in application layer

-- PostgreSQL pgvector index (if using PostgreSQL)
-- CREATE INDEX IF NOT EXISTS idx_clusters_embedding ON neural_clusters 
-- USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ============================================
-- Triggers for Updated At
-- ============================================

-- SQLite trigger for updating updated_at
CREATE TRIGGER IF NOT EXISTS trg_clusters_updated_at 
AFTER UPDATE ON neural_clusters
BEGIN
    UPDATE neural_clusters SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================
-- Views for Common Queries
-- ============================================

-- High-confidence clusters view
CREATE VIEW IF NOT EXISTS v_high_confidence_clusters AS
SELECT 
    id,
    theme,
    essence,
    confidence_score,
    cluster_size,
    access_count,
    created_at,
    updated_at
FROM neural_clusters
WHERE confidence_score >= 0.8
ORDER BY confidence_score DESC, cluster_size DESC;

-- Recently accessed clusters view
CREATE VIEW IF NOT EXISTS v_recently_accessed_clusters AS
SELECT 
    id,
    theme,
    essence,
    access_count,
    last_accessed,
    confidence_score
FROM neural_clusters
WHERE last_accessed IS NOT NULL
ORDER BY last_accessed DESC;

-- Cluster statistics view
CREATE VIEW IF NOT EXISTS v_cluster_statistics AS
SELECT 
    COUNT(*) as total_clusters,
    AVG(cluster_size) as avg_cluster_size,
    AVG(confidence_score) as avg_confidence,
    MAX(cluster_size) as max_cluster_size,
    MIN(cluster_size) as min_cluster_size,
    SUM(access_count) as total_accesses
FROM neural_clusters;

-- ============================================
-- Migration Notes
-- ============================================
-- For PostgreSQL deployments with pgvector:
-- 1. Ensure pgvector extension is installed
-- 2. Change embedding columns from BLOB to VECTOR(1536)
-- 3. Create vector similarity indexes
-- 4. Update queries to use vector operators (<=>, <#>, etc.)
--
-- For SQLite deployments:
-- 1. Keep embedding as BLOB
-- 2. Perform vector operations in application layer
-- 3. Consider using r-tree or fts5 for text search optimization
