# Story Vows Backend

REST API for managing weddings, guest uploads, payments, and live gallery walls.

## Architecture

- **Language**: Go 1.21+
- **Database**: PostgreSQL (pgx driver)
- **HTTP Framework**: Chi v5
- **Auth**: JWT (access + refresh tokens)
- **Payments**: Stripe (one-time, no subscriptions)
- **File Storage**: AWS S3 or Cloudflare R2 (S3-compatible)
- **Real-Time**: Server-Sent Events (SSE) for live wall broadcasts

## Project Structure

```
cmd/api/               # Main entrypoint
internal/
  ├── auth/            # JWT + password auth (signup, signin, refresh)
  ├── wedding/         # Wedding CRUD, QR codes, privacy controls
  ├── upload/          # Guest photo/video uploads, S3 integration
  ├── payment/         # Stripe checkout + webhook handling
  ├── gallery/         # Album, highlights, download endpoints
  ├── realtime/        # Live wall SSE hub
  ├── middleware/      # Auth, logging, CORS, rate-limiting
  ├── models/          # Domain models & DTOs
  ├── config/          # Environment configuration
  └── db/              # Database connection pooling
migrations/           # SQL migration files (001_init.up/down.sql)
```

## Setup (Local Development)

### Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Make (optional)

### 1. Clone & Install Dependencies

```bash
cd story-vows-backend
go mod download
```

### 2. Set Up Database

```bash
# Create a local PostgreSQL database
createdb storyvows_dev

# Run migrations (manual SQL)
psql -d storyvows_dev < migrations/001_init.up.sql
```

### 3. Configure Environment

```bash
cp .env.example .env
# Edit .env with your local database URL and secrets
```

Generate a JWT secret:

```bash
openssl rand -hex 32
```

### 4. Run the Server

```bash
go run ./cmd/api
```

Server starts on `http://localhost:8080`

**Health check:** `GET http://localhost:8080/health`

## API Endpoints

### Authentication (Public)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/signup` | Register new couple account |
| POST | `/api/auth/signin` | Sign in, receive JWT + refresh tokens |
| POST | `/api/auth/refresh` | Issue new access token using refresh token |
| POST | `/api/auth/signout` | Revoke refresh token |
| GET | `/api/auth/me` | Get authenticated user profile (requires auth) |

### Weddings (Authenticated)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/weddings` | Create a new wedding |
| GET | `/api/weddings` | List all weddings for user |
| GET | `/api/weddings/:id` | Get wedding details |
| PATCH | `/api/weddings/:id` | Update wedding (couple_names, venue, etc.) |
| DELETE | `/api/weddings/:id` | Soft-delete (deactivate) a wedding |
| PATCH | `/api/weddings/:id/privacy` | Set privacy mode (public/private/password-protected) |

### Uploads (Authenticated Couple + Guest)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/weddings/:id/uploads` | List all uploads for a wedding (owner) |
| PATCH | `/api/weddings/:id/uploads/:uploadId/approve` | Approve or reject upload |
| DELETE | `/api/weddings/:id/uploads/:uploadId` | Remove upload (owner) |
| POST | `/api/w/:slug/uploads` | Guest uploads photo/video (multipart, no auth) |

### Gallery (Authenticated)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/weddings/:id/album` | Get all approved uploads grouped by category |
| GET | `/api/weddings/:id/album/highlights` | Get curated highlights (random 20 photos) |
| GET | `/api/weddings/:id/album/download` | Download ZIP of all photos (Heritage/Legacy only) |
| GET | `/api/weddings/:id/wall` | SSE stream: live wall real-time updates |

### Guest Views (Public)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/w/:slug` | Get public wedding page (no auth) |
| POST | `/api/w/:slug/access` | Verify album password for protected weddings |

### Payments (Authenticated)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/checkout` | Create Stripe Checkout session (POST body: `{wedding_id, tier}`) |
| POST | `/api/webhooks/stripe` | Stripe webhook handler (no auth, signature verified) |

## Example cURL Requests

### Authentication

```bash
# Signup
curl -X POST "http://localhost:8080/api/auth/signup" \
  -H "Content-Type: application/json" \
  -d '{"email":"couple@example.com","password":"secure_password","full_name":"Julian & Sofia"}'

# Signin
curl -X POST "http://localhost:8080/api/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{"email":"couple@example.com","password":"secure_password"}'

# Refresh token
curl -X POST "http://localhost:8080/api/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<REFRESH_TOKEN>"}'
```

### Weddings

```bash
# Create wedding
curl -X POST "http://localhost:8080/api/weddings" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"couple_names":["Julian","Sofia"],"wedding_date":"2026-12-01","wedding_time":"18:30","venue":"The Willow House","address":"123 Meadow Lane","whatsapp_number":"+1234567890","ages":[25,27],"welcome_message":"Welcome!","template":"ivory-symphony","lighting":"Golden Hour Sunset","story_style":"Cinematic Movie","ceremony_style":"Kandyan (Sri Lankan)","venue_type":"Outdoor Garden","wedding_mood":"Romantic","wedding_theme":"Garden Wedding"}'

# Get wedding
curl -X GET "http://localhost:8080/api/weddings/<WEDDING_ID>" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Update wedding
curl -X PATCH "http://localhost:8080/api/weddings/<WEDDING_ID>" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"venue":"The Garden Venue","welcome_message":"See you soon!"}'

# Delete wedding
curl -X DELETE "http://localhost:8080/api/weddings/<WEDDING_ID>" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

### Generic Uploader (no wedding required)

```bash
# Upload file to a generic folder
curl -X POST "http://localhost:8080/api/uploads" \
  -F "folder_id=guest-uploads" \
  -F "file=@/path/to/photo.jpg"

# Upload file to default uploads folder
curl -X POST "http://localhost:8080/api/uploads" \
  -F "file=@/path/to/video.mp4"
```

### Invite Config CRUD

```bash
# Create invite config
curl -X POST "http://localhost:8080/api/weddings/<WEDDING_ID>/invite" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"couple":"Julian & Sofia","hashtag":"#JVWedding","intro":{"lines":["Welcome","Join us"],"tagline":"Celebrate with us","bg_image":"https://example.com/bg.jpg"},"story":{"title":"Our Story","events":[{"year":"2020","text":"We met"}]},"details":{"date":"2026-12-01","time":"5pm","venue":"The Willow House","address":"123 Meadow Lane","dress":"Formal"},"countdown":{"target_iso":"2026-12-01T17:00:00Z","label":"Big day"},"qr":{"title":"RSVP","subtitle":"Scan me","url":"https://example.com/rsvp"},"outro":{"line":"See you there","signature":"Julian & Sofia"}}'

# Get invite config
curl -X GET "http://localhost:8080/api/weddings/<WEDDING_ID>/invite" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Update invite config
curl -X PATCH "http://localhost:8080/api/weddings/<WEDDING_ID>/invite" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"hashtag":"#JulianSofia2026"}'

# Delete invite config
curl -X DELETE "http://localhost:8080/api/weddings/<WEDDING_ID>/invite" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

### Thank You Config CRUD

```bash
# Create thank you config
curl -X POST "http://localhost:8080/api/weddings/<WEDDING_ID>/thankyou" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"template":"ivory-symphony","couple":"Julian & Sofia","date":"2026-12-01","venue":"The Willow House","hashtag":"#JVWedding","hero_image":"https://example.com/hero.jpg","portrait":"https://example.com/portrait.jpg","intro":["Thank you for celebrating with us","Your presence meant everything"],"message":"We are grateful for your love and support.","signature":"Julian & Sofia","gallery":["https://example.com/1.jpg","https://example.com/2.jpg"],"closing":"With love"}'

# Get thank you config
curl -X GET "http://localhost:8080/api/weddings/<WEDDING_ID>/thankyou" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Update thank you config
curl -X PATCH "http://localhost:8080/api/weddings/<WEDDING_ID>/thankyou" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"message":"Thank you for sharing our special day."}'

# Delete thank you config
curl -X DELETE "http://localhost:8080/api/weddings/<WEDDING_ID>/thankyou" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

### WhatsApp Messaging

```bash
# Send plain text via Twilio or Meta, depending on config
curl -X POST "http://localhost:8080/api/whatsapp/send" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+94703842557","message":"Hello from Story Vows!"}'

# Explicit Twilio plain-text send
curl -X POST "http://localhost:8080/api/whatsapp/send-twilio" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+94703842557","message":"Hello from Story Vows!"}'

# Send Twilio WhatsApp template message
curl -X POST "http://localhost:8080/api/whatsapp/send-template" \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+94703842557","content_sid":"HXb5b62575e6e4ff6129ad7c8efe1f983e","content_variables":{"1":"12/1","2":"3pm"}}'
```

## Tier Pricing

One-time purchases (no subscriptions):

| Tier | Price | Features |
|------|-------|----------|
| **Elopement** | $199 | Up to 100 uploads, 1-year hosting, custom QR card |
| **Heritage** | $449 | Unlimited uploads, AI highlights, live wall, lifetime archive |
| **Legacy** | $799 | Multi-event, full concierge, RAW downloads, lifetime |

## Authentication Flow

### Signup / Signin

```javascript
// Signup
POST /api/auth/signup
{
  "email": "couple@example.com",
  "password": "secure_password_8chars_min",
  "full_name": "Julian & Sofia"
}
→ 201 {
    "access_token": "eyJhbGc...",
    "refresh_token": "uuid...",
    "user": { "id": "...", "email": "...", "full_name": "..." }
  }

// Use tokens
GET /api/weddings
  Authorization: Bearer eyJhbGc...
```

### Token Refresh

```javascript
POST /api/auth/refresh
{ "refresh_token": "uuid..." }
→ 200 { "access_token": "...", "refresh_token": "...", "user": {...} }
```

## Guest Upload Flow

1. **Guest scans QR** → lands on `/w/:slug` (frontend public page)
2. **Frontend resolves slug** → `GET /api/w/:slug` (no auth)
3. **Guest submits form** → `POST /api/w/:slug/uploads` (multipart)
   - File (photo/video)
   - Guest name (optional)
   - Wedding ID
4. **Backend validates** → checks tier limits, upload count, file type
5. **Stores in S3/R2** → generates public CDN URL
6. **Broadcasts to live wall** → SSE event sent to couple's venue screen

## Privacy Modes

- **Public**: Guests can view without password
- **Private**: Not accessible to guests (couple only)
- **Password-protected**: Guests enter password before viewing

Password is hashed server-side (bcrypt). Couple can change at any time.

## Payments & Stripe Integration

### Checkout Flow

```javascript
// Couple initiates purchase
POST /api/checkout
Authorization: Bearer ...
{
  "wedding_id": "uuid",
  "tier": "heritage"
}
→ 200 {
    "checkout_url": "https://checkout.stripe.com/pay/...",
    "session_id": "cs_test_..."
  }

// Couple redirected to Stripe Checkout
// On success → Stripe calls webhook
```

### Webhook Handler

Stripe sends `checkout.session.completed` event:

1. Validates webhook signature
2. Extracts `wedding_id`, `tier`, `payment_intent_id`
3. Updates `orders` table: status = "paid", paid_at = now()
4. Updates `weddings` table: tier = tier, expires_at = (1 year if Elopement)

## Configuration

All config via environment variables (see `.env.example`):

```bash
# Server
PORT=8080
ENV=production

# Database
DATABASE_URL=postgres://user:pass@host:5432/storyvows?sslmode=require

# JWT
JWT_SECRET=...
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Stripe
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...

# S3/R2
S3_ENDPOINT=https://account.r2.cloudflarestorage.com
S3_BUCKET=storyvows
S3_REGION=auto
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
S3_PUBLIC_BASE_URL=https://cdn.storyvows.com

# App
FRONTEND_URL=https://storyvows.com
MAX_UPLOAD_SIZE=52428800
```

## Docker Deployment

Build image:

```bash
docker build -t storyvows-backend .
```

Run with Docker Compose or Kubernetes:

```bash
docker run -e DATABASE_URL=... -e STRIPE_SECRET_KEY=... \
  -e JWT_SECRET=... \
  -p 8080:8080 \
  storyvows-backend
```

## Testing

### Manual API Testing (curl)

```bash
# Signup
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpass1234",
    "full_name": "Test User"
  }'

# Create wedding (replace TOKEN with access_token from signup)
curl -X POST http://localhost:8080/api/weddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "couple_names": "Alice & Bob",
    "wedding_date": "2024-06-15",
    "venue": "The Manor",
    "welcome_message": "Join us for our celebration"
  }'
```

### Postman / Insomnia

Import endpoints from [API documentation](#api-endpoints). Store tokens in collection variables.

## Performance & Scalability

- **Connection Pool**: 20 max concurrent PostgreSQL connections
- **Rate Limiting**: 200 requests per minute (global)
- **S3 Uploads**: Handled by AWS SDK with resumable transfer support
- **Real-Time**: SSE scales to hundreds of concurrent connections per server
- **Caching**: Add Redis for token blacklist (signout) and session storage if needed

## Future Enhancements

- [ ] AI photo categorization (AWS Rekognition / OpenAI Vision)
- [ ] Photo watermarking on download
- [ ] Video transcoding (H.264, multiple quality tiers)
- [ ] Analytics dashboard (upload counts, engagement)
- [ ] Photo printing integration
- [ ] Guest RSVP tracking
- [ ] Message board / guestbook
- [ ] Email notifications (upload approved, tier expires, etc.)

## Security Checklist

- [x] Password hashing (bcrypt)
- [x] JWT tokens (HMAC-SHA256)
- [x] CORS configuration
- [x] Rate limiting
- [x] Stripe webhook signature verification
- [x] Privacy checks (ownership verification)
- [ ] SQL injection prevention (use pgx parametrized queries ✓)
- [ ] HTTPS in production (add to frontend/infra config)
- [ ] Secrets in environment, never in code ✓
- [ ] Token expiry (15 min access, 30 day refresh)

## License

MIT
