CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE task_status AS ENUM ('new', 'in_progress', 'done', 'cancelled');

CREATE TABLE tasks (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    status       task_status NOT NULL DEFAULT 'new',
    scheduled_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
