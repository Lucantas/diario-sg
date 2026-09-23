CREATE TABLE entities (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       text        NOT NULL CHECK (kind IN ('cnpj', 'processo', 'contrato')),
    key        text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, key)
);

CREATE TABLE entity_links (
    entity_id   uuid        NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    source      text        NOT NULL,
    record_kind text        NOT NULL,
    record_id   text        NOT NULL,
    role        text        NOT NULL,
    certainty   text        NOT NULL CHECK (certainty IN ('exata', 'forte', 'fraca')),
    evidence    text        NOT NULL,
    linked_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, source, record_kind, record_id, role)
);
CREATE INDEX entity_links_record_idx ON entity_links (source, record_kind, record_id);

CREATE TABLE fetch_runs (
    id             uuid PRIMARY KEY,
    source         text        NOT NULL,
    requested_from date        NOT NULL,
    requested_to   date        NOT NULL,
    found          int         NOT NULL CHECK (found >= 0),
    stored         int         NOT NULL CHECK (stored >= 0),
    skipped        int         NOT NULL CHECK (skipped >= 0),
    failed         int         NOT NULL CHECK (failed >= 0),
    error          text        NOT NULL DEFAULT '',
    started_at     timestamptz NOT NULL,
    finished_at    timestamptz NOT NULL
);
CREATE INDEX fetch_runs_source_idx ON fetch_runs (source, finished_at DESC);

CREATE FUNCTION entity_key(kind text, normalized text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT AS $$
    SELECT CASE WHEN kind = 'contrato' THEN regexp_replace(normalized, '^0+(\d)', '\1') ELSE normalized END
$$;

CREATE FUNCTION link_certainty(kind text, key text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT AS $$
    SELECT CASE
        WHEN kind = 'cnpj' THEN 'exata'
        WHEN kind = 'contrato' AND key !~ '[A-Z]' THEN 'fraca'
        ELSE 'forte'
    END
$$;

CREATE FUNCTION link_diario_acts(gazette uuid) RETURNS void
LANGUAGE sql AS $$
    INSERT INTO entities (kind, key)
    SELECT DISTINCT ae.kind, entity_key(ae.kind, ae.normalized)
    FROM act_entities ae JOIN acts a ON a.id = ae.act_id
    WHERE a.gazette_id = gazette AND ae.kind IN ('cnpj', 'processo', 'contrato')
    ORDER BY 1, 2
    ON CONFLICT (kind, key) DO NOTHING;

    INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
    SELECT e.id, 'diario_prefeitura', 'ato', ae.act_id::text, 'mencionado', link_certainty(e.kind, e.key), min(ae.value)
    FROM act_entities ae
    JOIN acts a ON a.id = ae.act_id
    JOIN entities e ON e.kind = ae.kind AND e.key = entity_key(ae.kind, ae.normalized)
    WHERE a.gazette_id = gazette AND ae.kind IN ('cnpj', 'processo', 'contrato')
    GROUP BY e.id, e.kind, e.key, ae.act_id
    ON CONFLICT DO NOTHING;
$$;

SELECT link_diario_acts(id) FROM gazettes;
