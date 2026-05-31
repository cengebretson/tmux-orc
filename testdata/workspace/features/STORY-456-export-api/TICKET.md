# TICKET.md — STORY-456

## Summary

Add a CSV export endpoint for report data.

## Description

Users need to download their report data as CSV for use in external tools.
The endpoint should support filtering by date range and paginate large exports
using a cursor so downloads don't time out on large datasets.

## Acceptance Criteria

- [ ] `GET /reports/export?from=&to=&cursor=` returns a CSV file or next-cursor JSON
- [ ] Response is `Content-Type: text/csv` with correct UTF-8 encoding
- [ ] Rows are streamed — endpoint does not buffer the entire result in memory
- [ ] Cursor-based pagination works across requests (stateless cursor)
- [ ] Empty result set returns an empty CSV with headers only
- [ ] Authenticated endpoint — 401 if no valid token

## Links

- Ticket: https://stories.example.com/STORY-456
- PR: https://github.com/example/my-app/pull/88
