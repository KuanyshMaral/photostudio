# MWork user sync (internal)

## Endpoint

`POST /internal/mwork/users/sync`

## Auth

`Authorization: Bearer <MWORK_SYNC_TOKEN>`

Token is read from `MWORK_SYNC_TOKEN` (fallback: `PHOTO_STUDIO_INTERNAL_TOKEN`).

## Environment

- `MWORK_SYNC_TOKEN` (required in production)
- `MWORK_SYNC_ENABLED` (optional, default `true`)
- `MWORK_SYNC_ALLOWED_IPS` (optional comma-separated allowlist for internal networks)

## Request

```json
{
  "mwork_user_id": "b7f0a34c-1244-4c3a-b6c4-94c7c9d7c10b",
  "email": "user@example.com",
  "role": "model"
}
```

## Email matching strategy

- PhotoStudio normalizes email for sync by applying `strings.TrimSpace` + `strings.ToLower`.
- Search by email is case-insensitive (matches `LOWER(email)`), which aligns with the existing unique index on `LOWER(email)` in the users table.
- If you need stronger guarantees across all entry points, consider migrating to a dedicated normalized email column with a unique index and backfill plan (not part of this change).

## Responses

- `200 OK` for updates/links
- `201 Created` for new user

```json
{
  "data": {
    "id": 123,
    "mwork_user_id": "b7f0a34c-1244-4c3a-b6c4-94c7c9d7c10b",
    "email": "user@example.com",
    "role": "model"
  }
}
```

## Error codes

| HTTP | code | description |
| --- | --- | --- |
| 401 | AUTH_MISSING | missing Authorization header |
| 401/403 | AUTH_INVALID | invalid token or auth format |
| 400 | VALIDATION_ERROR | invalid request payload |
| 409 | CONFLICT | conflicting user state |
| 500 | INTERNAL_ERROR | internal error |

### Error response format

```json
{
  "error": {
    "code": "STRING_CODE",
    "message": "human readable",
    "details": {
      "field_errors": {
        "email": "must be a valid email"
      }
    }
  }
}
```

### Validation error example

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request fields",
    "details": {
      "field_errors": {
        "mwork_user_id": "must be a valid UUID",
        "role": "must be one of model, employer, agency, admin"
      }
    }
  }
}
```

### Auth error example

```json
{
  "error": {
    "code": "AUTH_INVALID",
    "message": "Invalid internal token"
  }
}
```

## Example curl

```bash
curl -X POST http://localhost:3001/internal/mwork/users/sync \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -d '{"mwork_user_id":"b7f0a34c-1244-4c3a-b6c4-94c7c9d7c10b","email":"user@example.com","role":"model"}'
```

## Local test with MWork

### Docker compose setup

- PhotoStudio runs on port `8090` and is reachable inside the Docker network as `http://photostudio_api:8090`.
- Example environment for MWork:

```env
PHOTOSTUDIO_BASE_URL=http://photostudio_api:8090
```

### Sync request example

```bash
curl -X POST http://localhost:8090/internal/mwork/users/sync \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -d '{"mwork_user_id":"b7f0a34c-1244-4c3a-b6c4-94c7c9d7c10b","email":"user@example.com","role":"model"}'
```

### Checklist

- Sync returns `200` or `201`.
- User exists in PhotoStudio DB with `mwork_user_id` set:
  - `SELECT id, email, mwork_user_id, mwork_role FROM users WHERE mwork_user_id = 'b7f0a34c-1244-4c3a-b6c4-94c7c9d7c10b';`
