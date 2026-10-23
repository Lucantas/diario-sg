ALTER TABLE notifications_sent
    ADD COLUMN status     text        NOT NULL DEFAULT 'sent' CHECK (status IN ('pending', 'sent')),
    ADD COLUMN claimed_at timestamptz NOT NULL DEFAULT now(),
    ALTER COLUMN sent_at DROP NOT NULL,
    ALTER COLUMN sent_at DROP DEFAULT;
