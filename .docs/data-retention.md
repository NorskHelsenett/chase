# Data Retention Policy

Chase uses a tiered data retention strategy to keep the SQLite database performant while preserving historical data. All cleanup runs automatically on application startup.

## Ping Data — Three-Tier Timeseries

Raw ping results are progressively aggregated into coarser summaries:

| Age | Granularity | Table | Rows per server |
|-----|-------------|-------|-----------------|
| 0–7 days | Every ping (~4/hour) | `ping_results` | ~672/week |
| 7–30 days | Hourly averages | `ping_hourly_summaries` | ~552/month |
| 30+ days | Daily averages | `ping_daily_summaries` | ~365/year |

Each summary stores: `total`, `successful`, `failed`, `avg_response_time`, `min_response_time`, `max_response_time`.

Aggregation runs at startup in a background goroutine:
1. Raw pings 7–30 days old are rolled into hourly summaries, then deleted
2. Hourly summaries older than 30 days are rolled into daily summaries, then deleted
3. Orphaned `ping_details` are cleaned up

## Screenshots

- **One row per server** — upserted, not appended
- Each row stores full-size PNG + a 480px thumbnail (generated at capture time)
- Failed screenshot attempts are stored as failure markers (`error/{status_code}` MIME type) and retried after 24 hours
- Duplicate rows from legacy behavior are deduplicated at startup

## Security Reports

- **One row per server** — only the latest report is kept
- Old duplicates are cleaned up at startup

## Other Tables

| Table | Retention |
|-------|-----------|
| `batch_job_stores` | 30 days after completion |
| `batch_result_stores` | Orphans deleted when parent job is removed |
| `notification_logs` | 90 days |
| `sessions` | Deleted on expiry |
| `geo_caches` | Refreshed every 7 days (background ticker) |

## VACUUM

`VACUUM` runs at startup after all cleanup to reclaim disk space from deleted BLOBs.

## Server Metadata

Favicon, site title, description, and og:image are extracted from the HTML response during pings (first 64KB of `<head>`) and stored directly on the `servers` table. This avoids depending on a full security scan for basic UI metadata. Security scan data overrides if it provides richer values.
