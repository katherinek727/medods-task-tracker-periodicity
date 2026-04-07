-- Each recurring instance points back to the template task that spawned it.
-- The template itself has parent_task_id = NULL.
-- ON DELETE CASCADE: deleting the template removes all its instances.
ALTER TABLE tasks
    ADD COLUMN parent_task_id UUID NULL REFERENCES tasks(id) ON DELETE CASCADE;

CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id)
    WHERE parent_task_id IS NOT NULL;
