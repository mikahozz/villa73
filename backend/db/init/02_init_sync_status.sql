-- Enum for sync frequencies
CREATE TYPE sync_frequency AS ENUM ('HOURLY', 'DAILY', 'WEEKLY', 'MONTHLY');

-- Enum for sync status
CREATE TYPE sync_status AS ENUM ('SYNCED', 'NOT_SYNCED', 'ERROR');

-- Main sync status table (renamed from sync_status to sync_entries)
CREATE TABLE sync_entries (
    id BIGSERIAL PRIMARY KEY,
    sync_type VARCHAR(50) NOT NULL,  -- e.g., 'SPOT_PRICE', 'WEATHER_DATA', etc.
    target_date DATE NOT NULL,
    frequency sync_frequency NOT NULL,
    status sync_status NOT NULL DEFAULT 'NOT_SYNCED',
    last_attempt TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata JSONB,  -- For any type-specific data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Composite unique constraint
    UNIQUE (sync_type, target_date)
);

-- Index for common queries
CREATE INDEX idx_sync_entries_type_date ON sync_entries(sync_type, target_date);
CREATE INDEX idx_sync_entries_status ON sync_entries(status);
