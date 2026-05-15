# Story Vows API Endpoints Reference

Base URL: `http://localhost:8080` (development) or deployed instance

## Authentication (Public)

### Sign Up
```
POST /api/auth/signup
Content-Type: application/json

{
  "email": "couple@example.com",
  "password": "secure_password_min_8_chars",
  "full_name": "Julian & Sofia"
}

→ 201 Created
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "550e8400-e29b-41d4-a716...",
  "user": {
    "id": "uuid",
    "email": "couple@example.com",
    "full_name": "Julian & Sofia",
    "created_at": "2024-05-15T10:30:00Z",
    "updated_at": "2024-05-15T10:30:00Z"
  }
}

→ 409 Conflict (email already exists)
→ 400 Bad Request (invalid input)
```

### Sign In
```
POST /api/auth/signin
Content-Type: application/json

{
  "email": "couple@example.com",
  "password": "secure_password_min_8_chars"
}

→ 200 OK (same as signup response)
→ 401 Unauthorized (invalid credentials)
```

### Refresh Token
```
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "550e8400-e29b-41d4-a716..."
}

→ 200 OK (new token pair)
→ 401 Unauthorized (expired or invalid token)
```

### Sign Out
```
POST /api/auth/signout
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "refresh_token": "550e8400-e29b-41d4-a716..."
}

→ 200 OK
{
  "message": "signed out"
}
```

### Get Profile
```
GET /api/auth/me
Authorization: Bearer {access_token}

→ 200 OK
{
  "id": "uuid",
  "email": "couple@example.com",
  "full_name": "Julian & Sofia",
  "created_at": "2024-05-15T10:30:00Z",
  "updated_at": "2024-05-15T10:30:00Z"
}

→ 401 Unauthorized (missing or invalid token)
→ 404 Not Found (user deleted)
```

---

## Weddings (Authenticated)

All endpoints require: `Authorization: Bearer {access_token}`

### Create Wedding
```
POST /api/weddings
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "couple_names": "Julian & Sofia",
  "wedding_date": "2024-06-15",
  "venue": "The Willow House, Tuscany",
  "welcome_message": "Join us for our special day!"
}

→ 201 Created
{
  "id": "wedding-uuid",
  "owner_id": "user-uuid",
  "couple_names": "Julian & Sofia",
  "wedding_date": "2024-06-15T00:00:00Z",
  "venue": "The Willow House, Tuscany",
  "welcome_message": "Join us for our special day!",
  "qr_slug": "a1b2c3d4",
  "qr_code_url": "data:image/png;base64,iVBORw0KGgoAAAANS...",
  "tier": "elopement",
  "privacy": "public",
  "is_active": true,
  "upload_count": 0,
  "created_at": "2024-05-15T10:30:00Z",
  "updated_at": "2024-05-15T10:30:00Z"
}

→ 400 Bad Request (invalid date format: use YYYY-MM-DD)
```

### List Weddings
```
GET /api/weddings
Authorization: Bearer {access_token}

→ 200 OK
[
  {
    "id": "wedding-uuid",
    "couple_names": "Julian & Sofia",
    "wedding_date": "2024-06-15T00:00:00Z",
    "venue": "...",
    "qr_slug": "a1b2c3d4",
    "tier": "elopement",
    "privacy": "public",
    "upload_count": 42,
    ...
  },
  ...
]
```

### Get Wedding
```
GET /api/weddings/{wedding_id}
Authorization: Bearer {access_token}

→ 200 OK (same object as create response)
→ 404 Not Found
→ 403 Forbidden (not owner)
```

### Update Wedding
```
PATCH /api/weddings/{wedding_id}
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "couple_names": "Julian & Sofia (Updated)",
  "wedding_date": "2024-06-20",
  "venue": "...",
  "welcome_message": "..."
}

→ 200 OK (updated object)
→ 403 Forbidden (not owner)
→ 404 Not Found
```

### Delete Wedding (Soft)
```
DELETE /api/weddings/{wedding_id}
Authorization: Bearer {access_token}

→ 200 OK
{
  "message": "wedding deleted"
}

→ 403 Forbidden (not owner)
```

### Set Privacy
```
PATCH /api/weddings/{wedding_id}/privacy
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "privacy": "password_protected",
  "password": "guest-password-123"
}

→ 200 OK
{
  "message": "privacy updated"
}

→ 400 Bad Request (privacy must be one of: public, private, password_protected)
→ 403 Forbidden (not owner)
```

---

## Uploads (Authenticated Couple + Public Guest)

### List Uploads (Couple Only)
```
GET /api/weddings/{wedding_id}/uploads
Authorization: Bearer {access_token}

→ 200 OK
[
  {
    "id": "upload-uuid",
    "wedding_id": "wedding-uuid",
    "guest_name": "Aunt Mary",
    "file_url": "https://cdn.storyvows.com/weddings/.../photo.jpg",
    "file_type": "photo",
    "mime_type": "image/jpeg",
    "size_bytes": 2048000,
    "category": "ceremony",
    "is_approved": true,
    "uploaded_at": "2024-05-15T14:22:00Z"
  },
  ...
]

→ 403 Forbidden (not owner)
```

### Approve/Reject Upload
```
PATCH /api/weddings/{wedding_id}/uploads/{upload_id}/approve
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "approved": true
}

→ 200 OK
{
  "message": "updated"
}
```

### Delete Upload
```
DELETE /api/weddings/{wedding_id}/uploads/{upload_id}
Authorization: Bearer {access_token}

→ 200 OK
{
  "message": "deleted"
}

→ 404 Not Found
```

### Guest Upload (Public)
```
POST /api/w/{slug}/uploads
Content-Type: multipart/form-data
(no authorization required)

Form data:
  - file: [binary file]
  - guest_name: "Aunt Mary" (optional)
  - wedding_id: "wedding-uuid"

→ 201 Created
{
  "id": "upload-uuid",
  "wedding_id": "...",
  "guest_name": "Aunt Mary",
  "file_url": "https://...",
  "file_type": "photo",
  "category": "other",
  "is_approved": true,
  "uploaded_at": "2024-05-15T14:22:00Z"
}

→ 413 Payload Too Large (file > MAX_UPLOAD_SIZE)
→ 415 Unsupported Media Type (invalid MIME)
→ 402 Payment Required (upload limit reached for tier)
```

---

## Gallery & Album (Authenticated)

### Get Album
```
GET /api/weddings/{wedding_id}/album
Authorization: Bearer {access_token}

→ 200 OK
{
  "ceremony": [
    { "id": "...", "file_url": "...", ... },
    ...
  ],
  "candid": [...],
  "dancing": [...],
  "family": [...]
}
```

### Get Highlights
```
GET /api/weddings/{wedding_id}/album/highlights
Authorization: Bearer {access_token}

→ 200 OK
[
  { "id": "...", "file_url": "...", "category": "..." },
  ... (up to 20 random approved photos)
]
```

### Download Album (ZIP)
```
GET /api/weddings/{wedding_id}/album/download
Authorization: Bearer {access_token}

→ 200 OK (binary ZIP file)
Content-Type: application/zip
Content-Disposition: attachment; filename="wedding-uuid-album.zip"

→ 402 Payment Required (Elopement tier can't download)
  (Only Heritage and Legacy tiers can bulk download)
```

### Live Wall (SSE)
```
GET /api/weddings/{wedding_id}/wall
Authorization: Bearer {access_token}

→ 200 OK (stream continues)
Content-Type: text/event-stream

event: connected
data: {}

event: upload
data: {"id":"...","file_url":"...","guest_name":"..."}

event: upload
data: {"id":"...","file_url":"..."}

(stream continues, one event per guest upload)
```

---

## Guest Access (Public, No Auth)

### View Wedding (Public)
```
GET /api/w/{slug}
(no authorization required)

→ 200 OK
{
  "id": "wedding-uuid",
  "couple_names": "Julian & Sofia",
  "wedding_date": "2024-06-15T00:00:00Z",
  "venue": "...",
  "welcome_message": "...",
  "qr_slug": "a1b2c3d4",
  "tier": "heritage",
  "privacy": "public"
}

→ 404 Not Found (wedding not active)
```

### Verify Album Password
```
POST /api/w/{slug}/access
Content-Type: application/json
(no authorization required)

{
  "password": "guest-password-123"
}

→ 200 OK (same as GET /api/w/{slug})
→ 401 Unauthorized (wrong password)
→ 403 Forbidden (wedding is private, no guest access)
```

---

## Payments (Authenticated)

### Create Checkout Session
```
POST /api/checkout
Content-Type: application/json
Authorization: Bearer {access_token}

{
  "wedding_id": "wedding-uuid",
  "tier": "heritage"
}

→ 200 OK
{
  "checkout_url": "https://checkout.stripe.com/pay/cs_test_...",
  "session_id": "cs_test_..."
}

→ 400 Bad Request (invalid tier)
```

### Stripe Webhook (Server-to-Server)
```
POST /api/webhooks/stripe
Content-Type: application/json

(Stripe sends)
{
  "type": "checkout.session.completed",
  "data": {
    "object": {
      "id": "cs_test_...",
      "metadata": {
        "wedding_id": "...",
        "tier": "heritage"
      },
      ...
    }
  },
  ...
}

Signature: stripe-signature header

→ 200 OK (webhook processed)
→ 400 Bad Request (invalid signature)
```

---

## Health Check

### Server Health
```
GET /health

→ 200 OK
{
  "status": "ok"
}
```

---

## Error Responses

All errors return JSON:

```json
{
  "error": "error_key_or_message",
  "message": "optional_detailed_message"
}
```

### Common HTTP Status Codes

- **200 OK**: Success
- **201 Created**: Resource created
- **400 Bad Request**: Invalid input or validation error
- **401 Unauthorized**: Missing or invalid auth token
- **402 Payment Required**: Tier limit reached
- **403 Forbidden**: Access denied (ownership check failed)
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource already exists (duplicate email)
- **413 Payload Too Large**: File exceeds max size
- **415 Unsupported Media Type**: Invalid file type
- **429 Too Many Requests**: Rate limit exceeded (200 req/min)
- **500 Internal Server Error**: Server error

---

## Authentication Pattern

All authenticated requests must include:

```
Authorization: Bearer {access_token}
```

Example:
```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  http://localhost:8080/api/weddings
```

---

## Rate Limiting

- **Global**: 200 requests per minute
- **Per IP**: Shared pool
- **Header**: `RateLimit-*` headers included in responses

---

## CORS

Allowed origins: `FRONTEND_URL` from environment  
Allowed methods: GET, POST, PATCH, DELETE, OPTIONS  
Allowed headers: Accept, Authorization, Content-Type  
Credentials: Supported

---

## Pagination (Future)

Currently all list endpoints return full results. Future versions may add:

```
?page=1&limit=50
```

---

Last Updated: May 15, 2026
