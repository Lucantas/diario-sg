-- Edições do Diário Oficial já indexadas.
CREATE TABLE gazettes (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    edition_number text        NOT NULL DEFAULT '',
    published_at   date        NOT NULL,
    source_url     text        NOT NULL,
    storage_path   text        NOT NULL,
    checksum       text        NOT NULL UNIQUE,
    indexed_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX gazettes_published_at_idx ON gazettes (published_at DESC);

-- Atos individuais, com índice de busca textual em português.
CREATE TABLE acts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gazette_id uuid NOT NULL REFERENCES gazettes (id) ON DELETE CASCADE,
    type       text NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL,
    position   int  NOT NULL,
    search     tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('portuguese', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('portuguese', body), 'B')
    ) STORED,
    UNIQUE (gazette_id, position)
);
CREATE INDEX acts_search_idx ON acts USING GIN (search);
CREATE INDEX acts_type_idx ON acts (type);

-- Inscrições em alertas (double opt-in).
CREATE TABLE subscriptions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             text        NOT NULL,
    query             text        NOT NULL,
    status            text        NOT NULL CHECK (status IN ('pending', 'active', 'cancelled')),
    confirm_token     text        NOT NULL UNIQUE,
    unsubscribe_token text        NOT NULL UNIQUE,
    created_at        timestamptz NOT NULL DEFAULT now(),
    confirmed_at      timestamptz
);
CREATE INDEX subscriptions_active_idx ON subscriptions (status) WHERE status = 'active';

-- Garante no máximo um alerta por inscrição por edição.
CREATE TABLE notifications_sent (
    subscription_id uuid        NOT NULL REFERENCES subscriptions (id) ON DELETE CASCADE,
    gazette_id      uuid        NOT NULL REFERENCES gazettes (id) ON DELETE CASCADE,
    sent_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (subscription_id, gazette_id)
);
