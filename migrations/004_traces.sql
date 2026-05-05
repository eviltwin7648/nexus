CREATE TABLE traces (
    id            TEXT PRIMARY KEY,
    question      TEXT NOT NULL,
    answer        TEXT,
    status        TEXT NOT NULL,
    error         TEXT,
    total_ms      INT,
    total_tokens  INT,
    estimated_cost_usd NUMERIC(10, 6),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trace_steps (
    id          BIGSERIAL PRIMARY KEY,
    trace_id    TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    iteration   INT NOT NULL,
    tool        TEXT NOT NULL,
    input       JSONB NOT NULL DEFAULT '{}',
    output_len  INT,
    duration_ms INT,
    tokens_used INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_traces_created_at    ON traces(created_at DESC);
CREATE INDEX idx_trace_steps_trace_id ON trace_steps(trace_id);
