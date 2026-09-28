ALTER TABLE bills ADD COLUMN law_kind text NOT NULL DEFAULT '';

DROP INDEX bills_law_idx;
CREATE INDEX bills_law_idx ON bills (law_kind, law_number, law_year) WHERE law_number > 0;
