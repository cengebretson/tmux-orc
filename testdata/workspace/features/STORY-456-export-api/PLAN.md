# PLAN.md — STORY-456

## Approach

Stream rows directly from the database query into the HTTP response writer using
Go's `encoding/csv` writer. Use a cursor that encodes the last-seen row ID so
pagination is stateless. Keep the handler thin — query logic lives in the report
repository.

## Steps

- [x] Add `ExportRows(ctx, filter, afterID) ([]ReportRow, error)` to report repo
- [x] Implement cursor encode/decode (`internal/export/cursor.go`)
- [x] Implement streaming CSV handler (`internal/handler/export.go`)
- [x] Wire endpoint into router behind auth middleware
- [x] Unit tests: cursor round-trip, empty result, last page detection
- [x] Integration test: paginate through 3 pages of fixture data

## Risk / Unknowns

- Memory pressure: confirmed streaming approach avoids buffering. DB query
  uses a server-side cursor (pgx `QueryRow` rows iterator), not `QueryAll`.
- Encoding: all string fields are sanitized to strip embedded newlines and
  quotes before writing — standard `encoding/csv` handles quoting.

## QA Notes

- Verify UTF-8 BOM is NOT added (Excel handles it fine without; some parsers break with it)
- Test with 0, 1, and 5001 rows to hit pagination boundary
- Check `X-Export-Next-Cursor` is absent on the last page
- Verify 401 with no token, 400 with invalid date format
