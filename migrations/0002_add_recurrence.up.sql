CREATE TYPE recurrence_type AS ENUM (
    'daily',
    'monthly',
    'specific_dates',
    'even_days',
    'odd_days'
);

ALTER TABLE tasks
    ADD COLUMN recurrence_type     recurrence_type NULL,
    ADD COLUMN recurrence_interval INT             NULL,
    ADD COLUMN recurrence_day      INT             NULL,
    ADD COLUMN recurrence_dates    TIMESTAMPTZ[]   NULL;

-- recurrence_interval must be >= 1 when type is daily
ALTER TABLE tasks
    ADD CONSTRAINT chk_recurrence_interval
        CHECK (recurrence_type != 'daily' OR recurrence_interval >= 1);

-- recurrence_day must be 1–30 when type is monthly
ALTER TABLE tasks
    ADD CONSTRAINT chk_recurrence_day
        CHECK (recurrence_type != 'monthly' OR (recurrence_day >= 1 AND recurrence_day <= 30));
