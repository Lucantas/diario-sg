CREATE TABLE bills (
    process_number    int         NOT NULL,
    process_year      int         NOT NULL,
    kind              text        NOT NULL,
    doc_label         text        NOT NULL,
    doc_number        int         NOT NULL,
    doc_year          int         NOT NULL,
    summary           text        NOT NULL,
    authors           text        NOT NULL,
    presented_on      date,
    status            text        NOT NULL,
    current_body      text        NOT NULL,
    last_movement     text        NOT NULL,
    source_updated_at timestamptz,
    law_number        int         NOT NULL,
    law_year          int         NOT NULL,
    law_url           text        NOT NULL,
    url               text        NOT NULL,
    fetched_at        timestamptz NOT NULL,
    PRIMARY KEY (process_number, process_year)
);

CREATE INDEX bills_kind_idx ON bills (kind);
CREATE INDEX bills_doc_idx ON bills (doc_number, doc_year);
CREATE INDEX bills_law_idx ON bills (law_number, law_year) WHERE law_number > 0;
CREATE INDEX bills_fetched_idx ON bills (fetched_at);
CREATE INDEX bills_search_idx ON bills USING gin (to_tsvector('portuguese_unaccent', summary || ' ' || authors));

CREATE TABLE bill_events (
    process_number int         NOT NULL,
    process_year   int         NOT NULL,
    position       int         NOT NULL,
    happened_at    timestamptz NOT NULL,
    label          text        NOT NULL,
    text           text        NOT NULL,
    sector         text        NOT NULL,
    PRIMARY KEY (process_number, process_year, position),
    FOREIGN KEY (process_number, process_year) REFERENCES bills ON DELETE CASCADE
);

CREATE TABLE bill_opinions (
    process_number int  NOT NULL,
    process_year   int  NOT NULL,
    position       int  NOT NULL,
    result         text NOT NULL,
    issued_on      date,
    committee      text NOT NULL,
    rapporteur     text NOT NULL,
    PRIMARY KEY (process_number, process_year, position),
    FOREIGN KEY (process_number, process_year) REFERENCES bills ON DELETE CASCADE
);
