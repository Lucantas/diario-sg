ALTER TABLE acts
    ADD COLUMN modality text,
    ADD COLUMN main_value_cents bigint;

CREATE INDEX acts_modality_idx ON acts (modality) WHERE modality IS NOT NULL;
