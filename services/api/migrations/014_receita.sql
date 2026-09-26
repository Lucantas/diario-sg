CREATE TABLE rf_companies (
    cnpj_base       text PRIMARY KEY,
    name            text   NOT NULL,
    legal_nature    text   NOT NULL,
    capital_cents   bigint NOT NULL,
    size            text   NOT NULL,
    reference_month date   NOT NULL
);

CREATE TABLE rf_establishments (
    cnpj               text PRIMARY KEY,
    cnpj_base          text    NOT NULL,
    headquarters       boolean NOT NULL,
    trade_name         text    NOT NULL,
    status             text    NOT NULL,
    status_since       date,
    status_reason      text    NOT NULL,
    opened_at          date,
    main_activity_code text    NOT NULL,
    main_activity      text    NOT NULL,
    other_activities   jsonb   NOT NULL,
    street             text    NOT NULL,
    number             text    NOT NULL,
    complement         text    NOT NULL,
    district           text    NOT NULL,
    zip                text    NOT NULL,
    city               text    NOT NULL,
    uf                 text    NOT NULL,
    reference_month    date    NOT NULL
);
CREATE INDEX rf_establishments_base_idx ON rf_establishments (cnpj_base);

CREATE TABLE rf_partners (
    cnpj_base       text     NOT NULL,
    kind            smallint NOT NULL CHECK (kind IN (1, 2, 3)),
    name            text     NOT NULL,
    document        text     NOT NULL,
    role            text     NOT NULL,
    since           date,
    reference_month date     NOT NULL
);
CREATE INDEX rf_partners_base_idx ON rf_partners (cnpj_base);
