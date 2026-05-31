# SPEC.md — STORY-456

## Context

The reporting dashboard shows data in-app but has no export path. Power users
export manually using copy-paste or screenshots, which is error-prone and slow.
A CSV endpoint lets them pull data directly into spreadsheets or data pipelines.

## Scope

### In scope
- `GET /reports/export` endpoint with `from`, `to`, `cursor` query params
- Streaming CSV response (no full buffering)
- Opaque cursor encoding date + row offset for stateless pagination
- Unit tests for cursor encoding/decoding and CSV row formatting
- Integration test for end-to-end export flow

### Out of scope
- Excel (`.xlsx`) format — CSV only for now
- Scheduled / async export jobs
- Export of non-report entities (users, settings, etc.)

## Behavior

**Request**
```
GET /reports/export?from=2026-01-01&to=2026-03-31&cursor=<opaque>
Authorization: Bearer <token>
```

**Response (has more rows)**
```
HTTP/1.1 200 OK
Content-Type: text/csv; charset=utf-8
X-Export-Next-Cursor: <opaque>

date,metric,value
2026-01-01,page_views,1420
...
```

**Response (last page)**  
No `X-Export-Next-Cursor` header. Empty body with headers if no rows.

**Cursor format (internal)**  
Base64-encoded JSON: `{ "after_id": 12345, "from": "2026-01-01", "to": "2026-03-31" }`

## Open Questions

- [ ] Max rows per page? Suggest 5000 — confirm with backend team.
