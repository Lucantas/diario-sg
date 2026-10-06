CREATE FUNCTION numeric_terms(t text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS
$$ SELECT coalesce(string_agg(m[1], ' '), '') FROM regexp_matches(t, '([0-9][0-9./-]*[0-9])', 'g') AS m $$;

ALTER TABLE acts ADD COLUMN numeric_terms text GENERATED ALWAYS AS (numeric_terms(body)) STORED;

CREATE INDEX acts_numeric_terms_trgm_idx ON acts USING gin (numeric_terms gin_trgm_ops);

DROP INDEX acts_body_trgm_idx;
