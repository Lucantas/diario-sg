CREATE TABLE federal_amendments (
    id               bigserial PRIMARY KEY,
    code             text   NOT NULL,
    year             int    NOT NULL,
    kind             text   NOT NULL,
    author           text   NOT NULL,
    number           text   NOT NULL,
    function         text   NOT NULL,
    subfunction      text   NOT NULL,
    program          text   NOT NULL,
    action           text   NOT NULL,
    committed_cents  bigint NOT NULL,
    liquidated_cents bigint NOT NULL,
    paid_cents       bigint NOT NULL
);

CREATE TABLE federal_amendment_payments (
    id           bigserial PRIMARY KEY,
    code         text   NOT NULL,
    author       text   NOT NULL,
    kind         text   NOT NULL,
    month        date   NOT NULL,
    cnpj         text   NOT NULL,
    name         text   NOT NULL,
    legal_nature text   NOT NULL,
    value_cents  bigint NOT NULL
);

CREATE INDEX federal_amendment_payments_cnpj_idx ON federal_amendment_payments (cnpj);

CREATE TABLE federal_transfers (
    id          bigserial PRIMARY KEY,
    month       date   NOT NULL,
    kind        text   NOT NULL,
    organ       text   NOT NULL,
    function    text   NOT NULL,
    program     text   NOT NULL,
    action      text   NOT NULL,
    label       text   NOT NULL,
    cnpj        text   NOT NULL,
    name        text   NOT NULL,
    value_cents bigint NOT NULL
);

CREATE INDEX federal_transfers_month_idx ON federal_transfers (month);
