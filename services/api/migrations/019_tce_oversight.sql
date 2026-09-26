CREATE TABLE tce_accounts (
    year        int  PRIMARY KEY,
    opinion     text NOT NULL,
    process     text NOT NULL,
    responsible text NOT NULL
);

CREATE TABLE tce_penalties (
    condemnation text PRIMARY KEY,
    process      text   NOT NULL,
    year         int    NOT NULL,
    value_cents  bigint NOT NULL,
    organ        text   NOT NULL,
    nature       text   NOT NULL,
    session_date date
);

CREATE TABLE tce_stalled_works (
    id              bigserial PRIMARY KEY,
    contract        text   NOT NULL,
    cnpj            text   NOT NULL,
    contractor      text   NOT NULL,
    organ           text   NOT NULL,
    function        text   NOT NULL,
    total_cents     bigint NOT NULL,
    paid_cents      bigint NOT NULL,
    stalled_at      date,
    started_at      date,
    stalled_for     text   NOT NULL,
    reason          text   NOT NULL,
    contract_status text   NOT NULL,
    funding         text   NOT NULL
);

CREATE INDEX tce_stalled_works_cnpj_idx ON tce_stalled_works (cnpj);
