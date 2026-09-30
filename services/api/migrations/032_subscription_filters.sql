ALTER TABLE subscriptions
    ADD COLUMN filter_source text NOT NULL DEFAULT '',
    ADD COLUMN filter_type   text NOT NULL DEFAULT '',
    ADD COLUMN filter_organ  text NOT NULL DEFAULT '',
    ADD COLUMN filter_theme  text NOT NULL DEFAULT '';
