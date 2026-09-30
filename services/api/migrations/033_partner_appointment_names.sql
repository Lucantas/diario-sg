CREATE MATERIALIZED VIEW partner_appointment_names AS
WITH companies AS (
    SELECT DISTINCT left(e.key, 8) AS cnpj_base
    FROM entities e
    JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = 'ato'
    JOIN acts a ON a.id::text = l.record_id
    WHERE e.kind = 'cnpj' AND a.type IN ('contrato', 'aditivo', 'dispensa', 'licitacao', 'licenca_ambiental')
), names AS (
    SELECT DISTINCT p.name
    FROM rf_partners p
    JOIN companies c ON c.cnpj_base = p.cnpj_base
    WHERE p.kind = 2 AND array_length(regexp_split_to_array(trim(p.name), '\s+'), 1) >= 3
)
SELECT n.name, a.id AS act_id, g.published_at
FROM names n
JOIN acts a ON a.type IN ('nomeacao', 'exoneracao')
    AND a.search @@ phraseto_tsquery('portuguese_unaccent', n.name)
JOIN gazettes g ON g.id = a.gazette_id;

CREATE UNIQUE INDEX partner_appointment_names_key ON partner_appointment_names (name, act_id);
