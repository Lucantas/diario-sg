CREATE TABLE cgu_sanctions (
    register     text   NOT NULL CHECK (register IN ('CEIS', 'CNEP')),
    code         text   NOT NULL,
    cnpj         text   NOT NULL,
    cnpj_base    text   NOT NULL,
    name         text   NOT NULL,
    category     text   NOT NULL,
    starts_at    date,
    ends_at      date,
    published_at date,
    process      text   NOT NULL,
    organ        text   NOT NULL,
    organ_uf     text   NOT NULL,
    sphere       text   NOT NULL,
    scope        text   NOT NULL,
    legal_basis  text   NOT NULL,
    fine_cents   bigint,
    first_seen   date   NOT NULL,
    last_seen    date   NOT NULL,
    PRIMARY KEY (register, code)
);

CREATE INDEX cgu_sanctions_base_idx ON cgu_sanctions (cnpj_base);
