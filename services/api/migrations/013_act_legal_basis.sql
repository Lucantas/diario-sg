ALTER TABLE acts
    ADD COLUMN legal_basis text[] NOT NULL DEFAULT '{}',
    ADD COLUMN declared_increase_bp integer NOT NULL DEFAULT 0;

CREATE INDEX acts_legal_basis_idx ON acts USING gin (legal_basis) WHERE legal_basis <> '{}';
CREATE INDEX acts_declared_increase_idx ON acts (gazette_id) WHERE declared_increase_bp > 0;
