-- Busca sem acentos: dicionário português + unaccent no tsvector, e índice
-- trigram para busca por substring (nomes, CNPJ, números de contrato).
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TEXT SEARCH CONFIGURATION portuguese_unaccent (COPY = portuguese);
ALTER TEXT SEARCH CONFIGURATION portuguese_unaccent
    ALTER MAPPING FOR hword, hword_part, word, asciihword, asciiword, hword_asciipart
    WITH unaccent, portuguese_stem;

-- unaccent() é STABLE; índices e colunas geradas exigem IMMUTABLE.
CREATE FUNCTION unaccent_immutable(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
    AS $$ SELECT public.unaccent('public.unaccent', $1) $$;

ALTER TABLE acts DROP COLUMN search;
ALTER TABLE acts ADD COLUMN search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('portuguese_unaccent', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('portuguese_unaccent', body), 'B')
) STORED;
CREATE INDEX acts_search_idx ON acts USING GIN (search);
CREATE INDEX acts_body_trgm_idx ON acts USING GIN (unaccent_immutable(body) gin_trgm_ops);
