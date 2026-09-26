CREATE TABLE political_agent_pay (
    body           text   NOT NULL CHECK (body IN ('prefeitura', 'camara')),
    month          date   NOT NULL,
    name           text   NOT NULL,
    name_key       text   NOT NULL,
    role           text   NOT NULL CHECK (role IN ('prefeito', 'vice_prefeito', 'secretario', 'procurador_geral', 'vereador')),
    office         text   NOT NULL,
    gross_cents    bigint NOT NULL,
    discount_cents bigint,
    net_cents      bigint,
    PRIMARY KEY (body, month, name_key, role, office)
);

CREATE INDEX political_agent_pay_name_idx ON political_agent_pay (name_key);

CREATE TABLE councillors (
    legislature        int  NOT NULL,
    name               text NOT NULL,
    name_key           text NOT NULL,
    parliamentary_name text NOT NULL,
    party              text NOT NULL,
    situation          text NOT NULL,
    PRIMARY KEY (legislature, name_key)
);
