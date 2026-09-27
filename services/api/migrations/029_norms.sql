CREATE TABLE norms (
    kind           text NOT NULL,
    number         int  NOT NULL,
    year           int  NOT NULL,
    suffix         text NOT NULL,
    author         text NOT NULL,
    summary        text NOT NULL,
    promulgated_on date,
    text_url       text NOT NULL,
    PRIMARY KEY (kind, number, year, suffix)
);

CREATE INDEX norms_search_idx ON norms USING gin (to_tsvector('portuguese_unaccent', summary || ' ' || author));
