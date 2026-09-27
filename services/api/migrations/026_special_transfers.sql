CREATE TABLE special_transfers (
    plan_id          bigint PRIMARY KEY,
    code             text   NOT NULL,
    year             int    NOT NULL,
    status           text   NOT NULL,
    author           text   NOT NULL,
    amendment        text   NOT NULL,
    area             text   NOT NULL,
    value_cents      bigint NOT NULL,
    committed_cents  bigint NOT NULL,
    paid_cents       bigint NOT NULL,
    last_paid_at     date,
    work_plan_status text   NOT NULL,
    execution_end    date,
    report_kind      text   NOT NULL,
    report_at        date,
    executed_cents   bigint NOT NULL,
    pending_cents    bigint NOT NULL
);

CREATE TABLE special_transfer_executors (
    plan_id     bigint NOT NULL REFERENCES special_transfers ON DELETE CASCADE,
    position    int    NOT NULL,
    cnpj        text   NOT NULL,
    name        text   NOT NULL,
    object      text   NOT NULL,
    value_cents bigint NOT NULL,
    PRIMARY KEY (plan_id, position)
);
