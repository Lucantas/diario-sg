CREATE TABLE fiscal_totals (
    year             int    PRIMARY KEY,
    period           int    NOT NULL CHECK (period BETWEEN 1 AND 6),
    committed_cents  bigint NOT NULL,
    liquidated_cents bigint NOT NULL,
    paid_cents       bigint NOT NULL,
    source_url       text   NOT NULL
);
