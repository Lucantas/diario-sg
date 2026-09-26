CREATE TABLE tce_staff (
    month              date   NOT NULL,
    unit               text   NOT NULL,
    situation          text   NOT NULL,
    grp                text   NOT NULL,
    headcount          int    NOT NULL,
    remuneration_cents bigint NOT NULL,
    PRIMARY KEY (month, unit, situation)
);
