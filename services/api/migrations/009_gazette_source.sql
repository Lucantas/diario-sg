ALTER TABLE gazettes
    ADD COLUMN source text NOT NULL DEFAULT 'diario_prefeitura'
    CHECK (source IN ('diario_prefeitura', 'diario_camara'));

CREATE INDEX gazettes_source_published_idx ON gazettes (source, published_at);

CREATE OR REPLACE FUNCTION link_diario_acts(gazette uuid) RETURNS void
LANGUAGE sql AS $$
    INSERT INTO entities (kind, key)
    SELECT DISTINCT ae.kind, entity_key(ae.kind, ae.normalized)
    FROM act_entities ae JOIN acts a ON a.id = ae.act_id
    WHERE a.gazette_id = gazette AND ae.kind IN ('cnpj', 'processo', 'contrato')
    ORDER BY 1, 2
    ON CONFLICT (kind, key) DO NOTHING;

    INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
    SELECT e.id, g.source, 'ato', ae.act_id::text, 'mencionado', link_certainty(e.kind, e.key), min(ae.value)
    FROM act_entities ae
    JOIN acts a ON a.id = ae.act_id
    JOIN gazettes g ON g.id = a.gazette_id
    JOIN entities e ON e.kind = ae.kind AND e.key = entity_key(ae.kind, ae.normalized)
    WHERE a.gazette_id = gazette AND ae.kind IN ('cnpj', 'processo', 'contrato')
    GROUP BY e.id, e.kind, e.key, ae.act_id, g.source
    ON CONFLICT DO NOTHING;
$$;
