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
| POST | `/tasks` | Create task (with optional recurrence) |
| GET | `/tasks` | List all tasks |
| GET | `/tasks/{id}` | Get task by ID |
| PUT | `/tasks/{id}` | Update task |
| DELETE | `/tasks/{id}` | Delete task |

---

## Recurrence Feature

When creating a task, you can optionally include a `recurrence` object.  
The system will automatically generate all recurring instances for **1 year** from the base `scheduled_at`.

### Recurrence Types

| Type | Description | Required fields |
|------|-------------|-----------------|
| `daily` | Every N days | `interval` (≥ 1) |
| `monthly` | Fixed day of month | `day_of_month` (1–30) |
| `specific_dates` | Only on listed dates | `dates` (array of timestamps) |
| `even_days` | Even calendar days of month | — |
| `odd_days` | Odd calendar days of month | — |

### Examples

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

**Monthly — every 15th of the month:**
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

The response is an array — the first element is the base task, followed by all generated instances.

---

## Design Decisions & Assumptions

### 1. Recurrence generates concrete task instances
Rather than storing a recurrence rule and computing dates on the fly at query time, each recurring occurrence is stored as a **separate task row** in the database. This approach:
- keeps queries simple and fast (no runtime expansion)
- allows each instance to be independently updated or cancelled
- makes the data model straightforward for other modules in the MIS to consume

**Trade-off:** a 1-year daily task generates ~365 rows. For this domain (medical staff tasks) that is entirely acceptable.

### 2. 1-year generation horizon
The recurrence window is fixed at **1 year from `scheduled_at`**. This is a reasonable default for medical workflows — schedules rarely need to be planned further ahead. It can be made configurable via an env variable if needed.

### 3. `day_of_month` capped at 30, not 31
Months have varying lengths (28–31 days). To avoid silent skipping of dates (e.g. Feb 31), the maximum allowed `day_of_month` is **30**, which exists in every month. Day 31 is excluded intentionally.

### 4. The base task is also stored
When recurrence is provided, the `scheduled_at` of the request becomes the **first occurrence** (the template task). All subsequent dates are generated after it. The API returns the full list so the client knows exactly what was created.

### 5. Updating a recurring task updates only that instance
`PUT /tasks/{id}` updates a single task row. It does not cascade to sibling instances. This mirrors how tools like Google Calendar handle "edit this event" vs "edit all events" — the simpler, safer default.

### 6. Status defaults to `new`
If `status` is omitted in the request body, it defaults to `new`.

---

## Project Structure

```
cmd/api/            → entrypoint
internal/
  domain/task/      → Task entity, Recurrence model, Repository interface
  usecase/          → business logic, recurrence expansion
  repository/       → PostgreSQL implementation
  transport/http/   → chi router, HTTP handlers
  infrastructure/   → DB connection pool
migrations/         → SQL migration files
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
