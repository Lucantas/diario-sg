CREATE TABLE procurements (
    list        text NOT NULL,
    id          int  NOT NULL,
    notice      text NOT NULL,
    process     text NOT NULL,
    process_key text NOT NULL,
    modality    text NOT NULL,
    criterion   text NOT NULL,
    opens_at    timestamp,
    object      text NOT NULL,
    status      text NOT NULL,
    url         text NOT NULL,
    PRIMARY KEY (list, id)
);

CREATE INDEX procurements_process_key_idx ON procurements (process_key);

CREATE TABLE procurement_contracts (
    position       int    PRIMARY KEY,
    procurement_id int    NOT NULL,
    notice         text   NOT NULL,
    process        text   NOT NULL,
    process_key    text   NOT NULL,
    modality       text   NOT NULL,
    object         text   NOT NULL,
    value_cents    bigint NOT NULL,
    supplier       text   NOT NULL,
    instrument     text   NOT NULL,
    document_url   text   NOT NULL
);

CREATE INDEX procurement_contracts_process_key_idx ON procurement_contracts (process_key);
