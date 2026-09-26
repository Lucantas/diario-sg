CREATE TABLE payments (
    source           text   NOT NULL,
    year             int    NOT NULL,
    month            int    NOT NULL CHECK (month BETWEEN 1 AND 12),
    unit             text   NOT NULL,
    commitment       text   NOT NULL,
    cnpj             text   NOT NULL,
    function         text   NOT NULL,
    committed_cents  bigint NOT NULL,
    liquidated_cents bigint NOT NULL,
    paid_cents       bigint NOT NULL,
    PRIMARY KEY (source, year, month, unit, commitment, cnpj)
);

CREATE INDEX payments_cnpj_idx ON payments (cnpj, year);
