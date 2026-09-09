# Update — H3 reference inputs (id-based image/audio/video refs)

## Problem

`minimax-h3` (and other model) reference inputs **failed when passed by upload id**:

1. `POST /api/v1/leonardo/upload-image|upload-audio` picked a *random* round-robin
   Leonardo account and uploaded the media there.
2. The later generation request picked a *different* account.
3. Id references were passed straight through to Leonardo with **no ownership
   check**, so Leonardo rejected the foreign account's asset with a generic
   `502 "An error occurred."` (~2s after submit).

URL references already worked, because each generation attempt downloads and
uploads the file under whichever account is actually generating.

## Change

Media uploaded through the API is now **persisted locally**, and any reference
by id is **re-hosted (re-uploaded) under the account that runs the generation**.
Id references and URL references now behave identically and always target the
correct account, with full pool rotation intact.

### New: local upload store — `internal/uploadstore/`

- On-disk store (`./uploads/` by default) that keeps the raw bytes + metadata
  (`kind`, `ext`, `content_type`, `original_filename`) for every upload.
- Index rebuilt on startup from `*.meta.json` files; survives restarts.
- API: `Open`, `Save`, `Load`, `Meta`, `Count` (nil-safe for tests/legacy).

### Server wiring

- `internal/handler/generation.go` — added `UploadStore *uploadstore.Store` to
  `Server`.
- `cmd/server/main.go` — opens the store under `./uploads`, sets it on the
  server, and registers the new video-upload route.

### Upload endpoints persist media

- `POST /api/v1/leonardo/upload-image`
- `POST /api/v1/leonardo/upload-audio`
- `POST /api/v1/leonardo/upload-video` (**new** — previously video had no
  multipart upload endpoint; only URL/remote fetch)

All three upload to Leonardo (unchanged contract/validation) **and** save a
local copy under the same returned id.

### Resolvers re-host local ids

`internal/handler/admin.go`:
- New `saveLocalUpload` + `rehostLocalUpload` helpers.
- `resolveLeonardoImageID`, `resolveLeonardoAudioRef`,
  `resolveLeonardoVideoRef`: when given an id found in the local store, the
  bytes are re-uploaded under the *current generation session* before submit.
- Foreign ids not in the local store (e.g. Leonardo `GENERATED` assets) keep the
  old pass-through behaviour.
- Audio re-host surfaces the detected duration (authoritative for H3's 15s
  aggregate limit), matching the existing URL-reference semantics.
- Per-attempt caching means each retry re-hosts under the newly chosen account,
  consistent with how URL refs already behave.

## Result

| Reference input | Before | After |
|---|---|---|
| image by **id** | 502 (foreign account) | works |
| image by **url** | works | works |
| audio/music by **id** | 502 | works |
| audio/music by **url** | works | works |
| video by **id** | 502 | works |
| video by **url** | works | works |

### Verified end-to-end (2026-09-09, minimax-h3)

- image id + audio id → succeeded (`6abd4b5b-...`)
- video id → submitted and re-hosted (`47e30b4a-...`)
- image id + audio id via URL refs → succeeded (`41a5c9fa-...`, `5ad59859-...`)

## Team notes / API contract (unchanged)

- Upload: `POST /api/v1/leonardo/upload-image|audio|video` (multipart `file`,
  optional `token_id`) → returns `image_id` / `audio_id` / `video_id`.
- Reference them in generation:
  ```json
  {
    "model": "minimax-h3-accelerated",
    "image_reference": [{ "image": { "id": "<image_id>" } }],
    "audio_reference": [{ "audio": { "id": "<audio_id>", "duration": 6.87 } }]
  }
  ```
- The id **must** come from this server's upload endpoints to be re-hostable.
  Any image/audio/video URL also works and is uploaded under the generating
  account automatically.
- H3 rules still apply: audio needs an image/video reference, ≤3 refs per type,
  ≤15s total audio/video refs, `id`-only video refs require `duration`.

## Tests / status

- `internal/uploadstore/` unit tests pass (`go test ./internal/uploadstore/`).
- `go build ./...` and `go vet ./...` clean.
- Pre-existing test failures remain (unrelated to this change, reproduced on a
  clean checkout): model-normalization tests in `internal/handler`
  (`TestNormalizeVideoModelIDSupportsMinimaxH3`, `TestPublicRequestLogModelUsesMinimaxH3`)
  and round-robin ordering tests in `internal/token`.
