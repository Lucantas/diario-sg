CREATE TABLE municipal_commitments (
    entity_id        int    NOT NULL,
    entity           text   NOT NULL,
    year             int    NOT NULL,
    commitment_id    bigint NOT NULL,
    number           text   NOT NULL,
    committed_on     date   NOT NULL,
    cnpj             text   NOT NULL,
    name             text   NOT NULL,
    object           text   NOT NULL,
    process_kind     text   NOT NULL,
    process          text   NOT NULL,
    modality         text   NOT NULL,
    committed_cents  bigint NOT NULL,
    liquidated_cents bigint NOT NULL,
    paid_cents       bigint NOT NULL,
    PRIMARY KEY (year, entity_id, commitment_id)
);

CREATE INDEX municipal_commitments_cnpj_idx ON municipal_commitments (cnpj, committed_on DESC);

CREATE TABLE municipal_totals (
    year             int    NOT NULL,
    entity_id        int    NOT NULL,
    entity           text   NOT NULL,
    committed_cents  bigint NOT NULL,
    liquidated_cents bigint NOT NULL,
    paid_cents       bigint NOT NULL,
    PRIMARY KEY (year, entity_id)
);
