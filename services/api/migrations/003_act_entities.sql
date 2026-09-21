-- Campos extraídos de cada ato por regex (CNPJ, valor, contrato, processo).
CREATE TABLE act_entities (
    act_id     uuid NOT NULL REFERENCES acts (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('cnpj', 'valor', 'contrato', 'processo')),
    value      text NOT NULL,
    normalized text NOT NULL,
    PRIMARY KEY (act_id, kind, normalized),
    CHECK (kind <> 'valor' OR normalized ~ '^[0-9]+$')
);
CREATE INDEX act_entities_lookup_idx ON act_entities (kind, normalized);
