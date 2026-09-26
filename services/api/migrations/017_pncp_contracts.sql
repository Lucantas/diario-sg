CREATE TABLE pncp_contracts (
    control_number text PRIMARY KEY,
    org_cnpj       text   NOT NULL,
    unit_name      text   NOT NULL,
    year           int    NOT NULL,
    sequence       int    NOT NULL,
    kind           text   NOT NULL,
    process        text   NOT NULL,
    number         text   NOT NULL,
    supplier_cnpj  text   NOT NULL,
    supplier_name  text   NOT NULL,
    object         text   NOT NULL,
    value_cents    bigint NOT NULL,
    signed_at      date,
    published_at   date,
    starts_at      date,
    ends_at        date
);

CREATE INDEX pncp_contracts_supplier_idx ON pncp_contracts (supplier_cnpj);
