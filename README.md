# Task Tracker — Periodicity Feature

A Go REST API for a medical task tracker with support for recurring tasks.  
Built on top of the [medods test task](https://github.com/medods/test-task-for-junior-backend-developer) base project.

---

## Quick Start

```bash
docker compose up --build
```

API is available at `http://localhost:8080`  
Swagger UI at `http://localhost:8080/swagger/`

> If postgres was previously started with an old schema, recreate the volume:
> ```bash
> docker compose down -v && docker compose up --build
> ```

---

## API

Base prefix: `/api/v1`

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tasks` | Create task (with optional recurrence) |
| `GET` | `/tasks` | List tasks (supports filters) |
| `GET` | `/tasks/{id}` | Get task by ID |
| `PUT` | `/tasks/{id}` | Update a single task instance |
| `DELETE` | `/tasks/{id}` | Delete a single task |
| `DELETE` | `/tasks/{id}/recurrences` | Delete all recurring instances of a template task |

### GET /tasks — query filters

All filters are optional and combinable.

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | `new` \| `in_progress` \| `done` \| `cancelled` |
| `from` | RFC3339 | `scheduled_at >= from` |
| `to` | RFC3339 | `scheduled_at <= to` |
| `parent_id` | UUID | All instances belonging to a template task |

**Example — all pending tasks in April 2026:**
```
GET /api/v1/tasks?status=new&from=2026-04-01T00:00:00Z&to=2026-04-30T23:59:59Z
```

**Example — all instances of a recurring task:**
```
GET /api/v1/tasks?parent_id=<template-uuid>
```

---

## Recurrence Feature

When creating a task, include a `recurrence` object.  
The system generates all recurring instances for **1 year** from `scheduled_at` and stores them as individual rows.

### Recurrence Types

| Type | Description | Required fields |
|------|-------------|-----------------|
| `daily` | Every N days | `interval` (≥ 1) |
| `monthly` | Fixed day of month | `day_of_month` (1–30) |
| `specific_dates` | Only on listed dates | `dates` (array of RFC3339 timestamps) |
| `even_days` | Even calendar days of month | — |
| `odd_days` | Odd calendar days of month | — |

### Request examples

**Daily — every 2 days:**
```json
POST /api/v1/tasks
{
  "title": "Patient rounds",
  "description": "Morning ward rounds",
  "status": "new",
  "scheduled_at": "2026-04-07T09:00:00Z",
  "recurrence": {
    "type": "daily",
    "interval": 2
  }
}
```

**Monthly — every 15th:**
```json
{
  "title": "Monthly reporting",
  "scheduled_at": "2026-04-15T10:00:00Z",
  "recurrence": {
    "type": "monthly",
    "day_of_month": 15
  }
}
```

**Specific dates:**
```json
{
  "title": "Surgery prep",
  "scheduled_at": "2026-04-10T08:00:00Z",
  "recurrence": {
    "type": "specific_dates",
    "dates": [
      "2026-05-01T08:00:00Z",
      "2026-06-15T08:00:00Z"
    ]
  }
}
```

**Even days:**
```json
{
  "title": "Inventory check",
  "scheduled_at": "2026-04-07T08:00:00Z",
  "recurrence": {
    "type": "even_days"
  }
}
```

The response is an array — the first element is the template task, followed by all generated instances.  
Every instance carries a `parent_task_id` pointing back to the template.

### Bulk-deleting a recurring series

To cancel all future instances without deleting the template:

```
DELETE /api/v1/tasks/{template-id}/recurrences
```

Response:
```json
{ "deleted": 182 }
```

---

## Design Decisions & Assumptions

### 1. Recurrence generates concrete task instances
Each recurring occurrence is stored as a **separate task row**. This keeps queries simple and fast, lets each instance be independently updated or cancelled, and makes the data model easy for other MIS modules to consume.

**Trade-off:** a 1-year daily task generates ~365 rows. Acceptable for medical staff scheduling.

### 2. `parent_task_id` links instances to their template
Every generated instance has `parent_task_id` set to the template's ID. The template itself has `parent_task_id = NULL`. A foreign key with `ON DELETE CASCADE` ensures instances are cleaned up if the template is deleted directly.

### 3. 1-year generation horizon
Fixed at **1 year from `scheduled_at`**. Medical schedules rarely need planning beyond that. Configurable via env variable if needed.

### 4. `day_of_month` capped at 30, not 31
Months have 28–31 days. To avoid silently skipping February or short months, the maximum is **30**, which exists in every month.

### 5. Updating a recurring task updates only that instance
`PUT /tasks/{id}` is single-instance. No cascade to siblings — mirrors the "edit this event" default in calendar apps.

### 6. Bulk-delete keeps the template
`DELETE /tasks/{id}/recurrences` removes all children but leaves the template row intact. This lets the user keep the original task record while clearing the generated schedule.

### 7. Status defaults to `new`
If `status` is omitted in the request body, it defaults to `new`.

### 8. Graceful shutdown
The server listens for `SIGTERM`/`SIGINT` and gives in-flight requests **15 seconds** to complete before exiting. Safe for containerised deployments.

---

## Project Structure

```
cmd/api/            → entrypoint (server + graceful shutdown)
internal/
  domain/task/      → Task entity, Recurrence model, ListFilter, Repository interface
  usecase/          → business logic, recurrence expansion
  repository/       → PostgreSQL implementation (dynamic filter queries)
  transport/http/   → chi router, HTTP handlers
  infrastructure/   → DB connection pool
migrations/         → SQL migration files (applied in order by docker-compose)
docs/               → OpenAPI 2.0 spec (served at /swagger/)
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | `postgres` | Postgres user |
| `DB_PASSWORD` | `postgres` | Postgres password |
| `DB_NAME` | `taskdb` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
