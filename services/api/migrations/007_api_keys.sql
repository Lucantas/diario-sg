CREATE TABLE api_keys (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash     text        NOT NULL UNIQUE,
    key_prefix   text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    revoked_at   timestamptz,
    last_used_at timestamptz
);

CREATE TABLE api_key_usage (
    key_id uuid NOT NULL REFERENCES api_keys (id) ON DELETE CASCADE,
    day    date NOT NULL,
    tool   text NOT NULL,
    calls  int  NOT NULL DEFAULT 0,
    PRIMARY KEY (key_id, day, tool)
);
