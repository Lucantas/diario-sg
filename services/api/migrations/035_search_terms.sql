CREATE FUNCTION search_number(q text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS
$$ SELECT coalesce((SELECT m[1] FROM regexp_matches(q, '([0-9][0-9./-]*[0-9]|[0-9])', 'g') AS m
                    ORDER BY length(m[1]) DESC LIMIT 1), '') $$;

CREATE FUNCTION search_words(q text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS
$$ SELECT btrim(regexp_replace(regexp_replace(q, '[0-9][0-9./,-]*', ' ', 'g'), '(^|\s)\S{1,2}(?=\s|$)', ' ', 'g')) $$;
