# Jiuin API Contract

Both PHP (primary) and Go (backup) implement this contract. Every JSON
response is `{ "code": <HTTP status>, "message": <string>, "data": ... }`.
The browser only calls these paths through the public origin.

| Method | Path | Data |
| --- | --- | --- |
| GET | `/health`, `/api/v1/health` | `{ status: "ok" }` |
| GET | `/ready`, `/api/v1/ready` | database/storage/FFmpeg/FFprobe readiness |
| GET | `/api/v1/site`, `/api/v1/site/info`, `/api/v1/site/copyright` | site configuration |
| GET | `/api/v1/status`, `/api/v1/links`, `/api/v1/resources` | ecosystem configuration |
| GET | `/api/v1/statistics` | public aggregate statistics |
| POST | `/api/v1/statistics/visit` | service-token protected statistics write |
| GET | `/api/v1/background/random` | configured background URL |
| GET | `/api/v1/music`, `/api/v1/music/{id}` | ready public music metadata |
| GET | `/media/music/{id}/cover`, `/full`, `/lite` | JPEG or Range-capable MP3 |
| POST | `/api/v1/admin/music/upload`, `/api/v1/music/upload` | administrator upload |
| GET | `/ws/online` | Go-only RFC 6455 online count socket |

The frontend currently consumes only the read routes, media routes, and
`/ws/online`; the administrator upload route is included as a versioned
operational contract rather than inferred as a frontend call. The single SQL
source is `internal/core/schema.sql`: Go embeds it into the binary and PHP
loads that same file when it bootstraps the database.

## Upload contract

`Authorization: Bearer <JIUIN_MUSIC_ADMIN_TOKEN>` and a non-empty
`Idempotency-Key` header are required. The multipart fields are `file`,
`title`, `artist`, with optional `album`, `albumArtist`, `genre`, and `year`.
The response contains `uploadId`, `taskId`, `musicId`, `state`, and
`idempotentReplay`.

Both implementations calculate the IDs as `<kind>_sha256(Idempotency-Key)`.
`upload_requests.idempotency_key` is unique and records the uploaded content
hash. A repeated key with the same body returns the existing IDs; a repeated
key with different content returns `409`. This makes the explicitly configured
Nginx upload fallback safe against duplicate music rows and duplicate tasks.

URLs returned in music metadata are always root-relative `/media/music/...`.
Neither implementation returns a filesystem path, hostname, loopback address,
or Go port.
