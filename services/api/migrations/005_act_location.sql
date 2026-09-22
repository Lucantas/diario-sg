ALTER TABLE acts
    ADD COLUMN page_start int,
    ADD COLUMN page_end   int,
    ADD COLUMN organ      text NOT NULL DEFAULT '';
CREATE INDEX acts_organ_idx ON acts (organ);
