CREATE TABLE oauth_clients (
    id            text        PRIMARY KEY,
    name          text        NOT NULL DEFAULT '',
    redirect_uris text[]      NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_used_at  timestamptz
);

CREATE INDEX oauth_clients_unused ON oauth_clients (created_at) WHERE last_used_at IS NULL;

CREATE TABLE oauth_codes (
    code_hash    text        PRIMARY KEY,
    client_id    text        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    redirect_uri text        NOT NULL,
    challenge    text        NOT NULL,
    resource     text        NOT NULL DEFAULT '',
    expires_at   timestamptz NOT NULL
);

CREATE INDEX oauth_codes_expires_at ON oauth_codes (expires_at);
