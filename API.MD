# VSITE API Reference

Base URL: `http://localhost:8080`

All responses share the envelope below. `code` is `0` on success; non-zero means an error and `message` explains it. HTTP status codes are also set (200 / 400 / 401 / 403 / 404 / 500).

```json
{ "code": 0, "message": "ok", "data": { ... } }
```

- Public endpoints (front site) need no authentication.
- Admin endpoints require the header `X-Admin-Token: <token>` returned by `POST /api/admin/login` (`Authorization: Bearer <token>` is also accepted). Sessions expire after **12 hours**.
- First login with the default account (`ADMIN` / `123456`) forces a password change: until `POST /api/admin/password` succeeds, every admin endpoint except `/api/admin/password` and `/api/admin/logout` responds `403` with `{ "code": 403, "message": "Please change your password before using the admin panel", "need_change_password": true }`.

---

## Public: Videos

### GET /api/videos

List videos. Default order is the "random waterfall": the first page starts at a random point and the `cursor` walks forward, wrapping around the end — so infinite scrolling keeps working and does not repeat items (the client dedupes by `id` until `total` is reached).

Query parameters:

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| `page_size` | int | 12 | Items per page (max 60) |
| `cursor` | float | — | Cursor from the previous response's `next_cursor`. When present, random ordering is used |
| `blogger_id` | int | — | Only videos of this creator |
| `keyword` | string | — | Matches title / description / creator account |
| `order` | string | `random` | `latest` uses id-descending + `page` offset paging instead |
| `page` | int | 1 | Page number (only with `order=latest`) |
| `client_id` | string | — | Client id; when given, each item gets a `liked` flag |

Response `data`:

```json
{
  "items": [
    {
      "id": 21,
      "blogger_id": 5,
      "title": "Seaside sunrise timelapse",
      "description": "This video is all about ... — watch the full process!",
      "cover_url": "https://example.com/cover.jpg",
      "video_url": "https://example.com/video.mp4",
      "duration": "",
      "width": 0,
      "height": 0,
      "likes": 1234,
      "comments": 5,
      "views": 9999,
      "status": 1,
      "liked": false,
      "created_at": "2026-09-29 14:28:22",
      "blogger": { "id": 5, "account": "Fit With Zhang", "description": "...", "avatar": "https://...", "status": 1 }
    }
  ],
  "has_more": true,
  "next_cursor": "0.7312",
  "total": 20
}
```

Notes: in random mode `next_cursor` is the value to pass as `cursor` next time; when the end is reached it wraps to `"0"` and `has_more` stays `true` (stop scrolling once loaded unique items ≥ `total`).

### GET /api/videos/{id}

Single video. Optional `client_id` query to include `liked`. Response `data` is one video object as above, plus `latest_comments` (up to 5 most recent).

Returns `404` if not found.

### POST /api/videos/{id}/view

Increment view counter by 1. Returns `{ "video_id": id }`.

### POST /api/videos/{id}/like

Toggle like for a client. Body:

```json
{ "client_id": "c_abc123" }
```

Response `data`: `{ "video_id": 21, "liked": true, "likes": 1235 }`

### GET /api/videos/{id}/comments

Comments, newest first. Query: `page` (default 1), `page_size` (default 20).

Response `data`: `{ "items": [ { "id": 3, "video_id": 21, "nickname": "TestUser", "content": "...", "created_at": "2026-09-29 14:30:00" } ], "total": 1, "page": 1, "page_size": 10 }`

### POST /api/videos/{id}/comments

Post a comment. Body:

```json
{ "nickname": "TestUser", "content": "Great video!" }
```

- `content` is required, max 200 chars; `nickname` optional (default `Guest`), max 24 chars.
- Response `data`: the created comment object.
- Errors: `400` missing content / too long, `404` video not found.

---

## Public: Creators

### GET /api/bloggers

Active creators, newest first. Query: `page_size` (default 20, max 100).

```json
{ "items": [ { "id": 5, "account": "Fit With Zhang", "description": "...", "avatar": "https://...", "status": 1, "video_count": 4, "created_at": "..." } ], "total": 5 }
```

### GET /api/bloggers/{id}

Creator detail (same shape as one item above). `404` if not found.

---

## Admin: Auth

### POST /api/admin/login

```json
{ "username": "ADMIN", "password": "123456" }
```

Response `data`:

```json
{ "token": "<hex>", "expires_in": 43200, "username": "ADMIN", "must_change_password": true }
```

Errors: `401` incorrect username or password.

### POST /api/admin/logout

Invalidates the token (header `X-Admin-Token` or `?token=`).

### GET /api/admin/profile

Current admin. Response `data`: `{ "id": 5, "username": "ADMIN", "must_change_password": false, "created_at": "..." }`

### POST /api/admin/password

Change password (and optionally the username). Body:

```json
{ "old_password": "123456", "new_password": "newSecret1", "username": "ADMIN" }
```

- `new_password` required, min 6 chars, must differ from `old_password`.
- While `must_change_password` is true the old password is NOT verified; afterwards it IS verified (`401` if wrong).
- On success the current session is invalidated — sign in again.

---

## Admin: Creators

### GET /api/admin/bloggers

Query: `page` (default 1), `page_size` (default 20, max 100), `keyword` (matches account).

Response `data`: `{ "items": [...], "total": 5, "page": 1, "page_size": 20 }` (items include hidden creators and `video_count`).

### POST /api/admin/bloggers

```json
{ "account": "Travel Diaries", "description": "Road trips around the world", "avatar": "https://..." }
```

- `account` required, ≤ 64 chars, must be unique (`400` if duplicate).
- `description` ≤ 500 chars, `avatar` optional URL.

### PUT /api/admin/bloggers/{id}

Update. Body same as create plus `"status": 1` (1 = active, 0 = hidden). `404` if not found.

### DELETE /api/admin/bloggers/{id}

Deletes the creator **and all their videos/comments/likes**. `404` if not found.

---

## Admin: Videos

### GET /api/admin/videos

Query: `page`, `page_size`, `keyword` (matches title / creator account), `blogger_id`.

Includes offline videos. Response `data`: `{ "items": [...], "total": n, "page": 1, "page_size": 10 }`.

### POST /api/admin/videos

```json
{
  "blogger_id": 5,
  "title": "Sunrise timelapse",
  "video_url": "https://example.com/video.mp4",
  "cover_url": "https://example.com/cover.jpg",
  "description": "Shot at 5am",
  "status": 1
}
```

Validation (`400` on failure): `blogger_id` must exist; `title` required; `video_url` required and must start with `http(s)://` or `/`; `cover_url` (optional) same rule. Optional `duration` (string), `width`, `height` (ints).

### PUT /api/admin/videos/{id}

Update, body same as create (`status`: 1 online / 0 offline). `404` if not found.

### DELETE /api/admin/videos/{id}

Deletes the video and its comments/likes.

---

## Admin: Comments

### DELETE /api/admin/comments/{id}

Delete a comment; the parent video's comment counter is decremented. `404` if not found.

> Listing comments is done through `GET /api/admin/videos` + `GET /api/videos/{id}/comments` (the admin UI does exactly this).

---

## Admin: Upload

### POST /api/admin/upload

`multipart/form-data`, field `file`. Allowed types: `.jpg .jpeg .png .gif .webp .mp4 .webm .m3u8 .mov`; max 200 MB.

Response `data`: `{ "url": "/static/uploads/20260929143000-123456.jpg", "name": "...", "size": 10240 }`

Use the returned `url` directly as `cover_url` / `video_url`.

---

## Admin: Stats

### GET /api/admin/stats

Response `data`:

```json
{ "videos": 20, "bloggers": 5, "views": 1100000, "likes": 47900, "comments": 0 }
```

---

## Quick start with curl

```bash
BASE=http://127.0.0.1:8080

# 1. Login
TOKEN=$(curl -s -X POST $BASE/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"ADMIN","password":"123456"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['token'])")

# 2. (First login) change the default password
curl -s -X POST $BASE/api/admin/password \
  -H "X-Admin-Token: $TOKEN" -H 'Content-Type: application/json' \
  -d '{"old_password":"123456","new_password":"newSecret1"}'
# → login again to get a fresh token

# 3. Create a creator
curl -s -X POST $BASE/api/admin/bloggers \
  -H "X-Admin-Token: $TOKEN" -H 'Content-Type: application/json' \
  -d '{"account":"Travel Diaries","description":"Road trips"}'

# 4. Add a video for the creator
curl -s -X POST $BASE/api/admin/videos \
  -H "X-Admin-Token: $TOKEN" -H 'Content-Type: application/json' \
  -d '{"blogger_id":1,"title":"Sunrise timelapse","video_url":"https://example.com/v.mp4","cover_url":"https://example.com/c.jpg"}'

# 5. Public feed (random start, cursor paging)
curl -s "$BASE/api/videos?page_size=12"
```

Or run the full automated suite: `bash scripts/test_api.sh` (16 checks, end to end).
