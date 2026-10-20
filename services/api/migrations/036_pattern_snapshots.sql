CREATE TABLE pattern_snapshots (
    id          smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    fingerprint text NOT NULL,
    computed_at timestamptz NOT NULL DEFAULT now(),
    payload     bytea NOT NULL
);
