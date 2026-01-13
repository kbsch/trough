-- River job queue tables
-- These are the minimum tables needed for River to function.
-- For the full migration, you can run: river migrate-up --database-url $DATABASE_URL

CREATE TABLE IF NOT EXISTS river_job (
    id bigserial PRIMARY KEY,
    state text NOT NULL DEFAULT 'available',
    attempt smallint NOT NULL DEFAULT 0,
    max_attempts smallint NOT NULL,
    attempted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    finalized_at timestamptz,
    scheduled_at timestamptz NOT NULL DEFAULT NOW(),
    priority smallint NOT NULL DEFAULT 1,
    args jsonb NOT NULL,
    attempted_by text[],
    errors jsonb[],
    kind text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}',
    queue text NOT NULL DEFAULT 'default',
    tags text[] NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS river_job_state_and_finalized_at_index ON river_job (state, finalized_at) WHERE finalized_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS river_job_prioritized_fetching_index ON river_job (state, queue, priority, scheduled_at, id) WHERE state IN ('available', 'retryable');
CREATE INDEX IF NOT EXISTS river_job_args_index ON river_job USING gin (args);
CREATE INDEX IF NOT EXISTS river_job_kind ON river_job (kind);
CREATE INDEX IF NOT EXISTS river_job_metadata_index ON river_job USING gin (metadata);

CREATE TABLE IF NOT EXISTS river_leader (
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text PRIMARY KEY
);

CREATE INDEX IF NOT EXISTS river_leader_name ON river_leader (name);

CREATE TABLE IF NOT EXISTS river_queue (
    name text PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    metadata jsonb NOT NULL DEFAULT '{}',
    paused_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT NOW()
);
