CREATE TABLE error_reports (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gazette_id uuid        NOT NULL REFERENCES gazettes (id) ON DELETE CASCADE,
    position   int         NOT NULL CHECK (position >= 0),
    act_title  text        NOT NULL,
    kind       text        NOT NULL CHECK (kind IN ('texto_errado', 'tipo_errado', 'orgao_errado', 'pagina_errada', 'outro')),
    message    text        NOT NULL DEFAULT '',
    status     text        NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'resolvido', 'descartado')),
    created_at timestamptz NOT NULL DEFAULT now(),
    closed_at  timestamptz
);
CREATE INDEX error_reports_open_idx ON error_reports (created_at) WHERE status = 'aberto';
