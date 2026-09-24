ALTER TABLE subscriptions
    ALTER COLUMN query DROP NOT NULL,
    ADD COLUMN entity_kind  text CHECK (entity_kind IN ('cnpj', 'processo', 'contrato')),
    ADD COLUMN entity_key   text,
    ADD COLUMN entity_label text,
    ADD CONSTRAINT subscriptions_query_or_entity CHECK (
        (query IS NOT NULL AND entity_kind IS NULL AND entity_key IS NULL AND entity_label IS NULL)
        OR (query IS NULL AND entity_kind IS NOT NULL AND entity_key IS NOT NULL AND entity_label IS NOT NULL)
    );
