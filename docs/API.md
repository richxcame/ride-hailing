# Ride Hailing Platform API

Comprehensive reference for every HTTP surface exposed by the ride-hailing platform microservices. Each section below reflects the current code in `main` and is grouped by service so you can map handlers in `internal/<service>` to the contract documented here.

> **Documentation strategy**
> This file stays the canonical index so new contributors can see the entire surface at a glance. If/when individual sections start to grow beyond what is manageable, we can graduate them into `docs/api/<service>.md` but keep this file as the table of contents.

## Quick Navigation

- [Service matrix](#service-matrix)
- [Conventions](#conventions)
- [Auth Service (:8081)](#auth-service-8081)
- [Rides Service (:8082)](#rides-service-8082)
- [Mobile API (:8087)](#mobile-api-8087)
- [Geo Service (:8083)](#geo-service-8083)
- [Payments Service (:8084)](#payments-service-8084)
- [Promos Service (:8089)](#promos-service-8089)
- [Notifications Service (:8085)](#notifications-service-8085)
- [Real-time Service (:8086)](#real-time-service-8086)
- [Admin Service (:8088)](#admin-service-8088)
- [Analytics Service (:8091)](#analytics-service-8091)
- [Fraud Service (:8092)](#fraud-service-8092)
- [ML ETA Service (:8093)](#ml-eta-service-8093)
- [Scheduler Service (:8090)](#scheduler-service-8090)
- [Health, metrics & observability](#health-metrics--observability)
## Service Matrix

| Service | Port | Base path (local) | Notes |
| --- | --- | --- | --- |
| Auth | 8081 | `http://localhost:8081/api/v1/auth` | Registration, login, profile management, JWT minting |
| Rides | 8082 | `http://localhost:8082/api/v1` | Ride lifecycle for riders & drivers, surge lookup, rate limiting enabled |
| Geo | 8083 | `http://localhost:8083/api/v1/geo` | Driver location updates + distance/ETA helpers |
| Payments | 8084 | `http://localhost:8084/api/v1` | Wallets, ride payments, refunds, Stripe webhook |
| Notifications | 8085 | `http://localhost:8085/api/v1` | Push/SMS/email notifications + ride lifecycle events |
| Real-time | 8086 | `http://localhost:8086/api/v1` | WebSocket gateway, chat history, driver tracking |
| Mobile API | 8087 | `http://localhost:8087/api/v1` | Mobile-friendly aggregates: ride history, receipts, favorites, profile |
| Admin | 8088 | `http://localhost:8088/api/v1/admin` | Admin dashboard, user/driver governance, ride stats |
| Promos | 8089 | `http://localhost:8089/api/v1` | Ride types, fare calc, promo/referral workflows |
| Scheduler | 8090 | `http://localhost:8090` | Background worker; only exposes `/healthz` and `/metrics` |
| Analytics | 8091 | `http://localhost:8091/api/v1/analytics` | Admin-only BI endpoints |
| Fraud | 8092 | `http://localhost:8092/api/v1/fraud` | Admin-only fraud alerts & risk operations |
| ML ETA | 8093 | `http://localhost:8093/api/v1/eta` | Public ETA prediction + admin ML controls |

When fronted by Kong or Istio, these ports collapse behind the gateway but the path structure (`/api/v1/...`) remains identical.
## Conventions

### Authentication & Roles

- All protected endpoints expect `Authorization: Bearer <jwt>` using the token issued by the Auth service. Tokens encode the `user_id` and `role` claims consumed by the shared Gin middleware.
- Roles defined in `pkg/models/user.go`:
  - `rider`: Standard consumer accounts.
  - `driver`: Supply-side accounts.
  - `admin`: Back-office or service accounts. Every admin endpoint below requires this role.
- Service-to-service calls (e.g., rides → notifications) also go through the same middleware. Use a service account seeded as `admin` when invoking internal-only endpoints.

### Response Envelope

Every handler uses the helpers in `pkg/common/response.go`, so responses are wrapped consistently:

```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "limit": 20,
    "offset": 0,
    "total": 42
  }
}
```

Errors adopt the same envelope:

```json
{
  "success": false,
  "error": {
    "code": 401,
    "message": "unauthorized"
  }
}
```

When an `AppError` is returned you will receive the HTTP status code it specifies (e.g., 400, 404, 409, etc.).

### Pagination & Filtering

Two pagination styles exist:

| Pattern | Query params | Used by |
| --- | --- | --- |
| Page-based | `page` (default 1), `per_page` (default 10, max 100) | `GET /api/v1/rides`, `GET /api/v1/fraud/alerts` |
| Offset-based | `limit` (default 20, max 100 unless stated), `offset` (default 0) | Wallet transactions, notifications, ride history, analytics listings, etc. |

Date filters follow `YYYY-MM-DD`. Unless otherwise noted, the backend interprets them as UTC and expands `end_date` to the end of the day.

### Rate Limiting

The rides service has Redis-backed token buckets enabled by default:

- Authenticated users: 120 requests/min with a 40-request burst.
- Anonymous requests (only applicable to public endpoints): 60 requests/min with a 20-request burst.

Responses include the standard headers emitted by `pkg/middleware/rate_limit.go`:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`
- `X-RateLimit-Resource` (the logical endpoint key)

### IDs, Numbers & Times

- Identifiers are UUIDv4 strings. When a payload accepts an ID it expects the canonical string form (e.g., `a3d2...`).
- Monetary values are floating point numbers in your configured currency (USD by default).
- Timestamps are RFC 3339 strings (`2025-01-01T12:00:00Z`) in UTC.
## Service Reference
### Auth Service (:8081)

Handles user onboarding, authentication and profile edits. Routes are defined in `internal/auth/handler.go`.

- **Base URL:** `http://localhost:8081/api/v1/auth`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/register` | None | Create a rider or driver account. Mirrors `models.RegisterRequest`. |
| POST | `/login` | None | Exchanges credentials for a JWT + user payload. |
| GET | `/profile` | Bearer | Returns the current user (`models.User`). |
| PUT | `/profile` | Bearer | Updates `first_name`, `last_name`, `phone_number`. |

#### Example: POST /api/v1/auth/register

```json
{
  "email": "rider@example.com",
  "password": "Sup3rSecure!",
  "phone_number": "+15551230000",
  "first_name": "Riley",
  "last_name": "Chen",
  "role": "rider"
}
```

Success → `201 Created` with the stored `models.User` (password hash omitted).

#### Example: POST /api/v1/auth/login

```json
{
  "email": "rider@example.com",
  "password": "Sup3rSecure!"
}
```

Response:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "5a4e...",
      "email": "rider@example.com",
      "first_name": "Riley",
      "last_name": "Chen",
      "role": "rider"
    },
    "token": "eyJhbGciOi..."
  }
}
```

#### Example: PUT /api/v1/auth/profile

```json
{
  "first_name": "Riley",
  "last_name": "Chen",
  "phone_number": "+15559870000"
}
```

Validates presence of all three fields. On success the updated user object is returned.
### Rides Service (:8082)

`internal/rides/handler.go` exposes the complete ride lifecycle and enforces role-based access. All endpoints sit under `http://localhost:8082/api/v1` and require a valid JWT. Rate limiting is active on every route registered through `handler.RegisterRoutes`.

**Ride status values:** `requested`, `accepted`, `in_progress`, `completed`, `cancelled` (see `pkg/models/ride.go`).

#### Rider endpoints

| Method | Path | Description |
| --- | --- | --- |
| POST | `/rides` | Create a ride using `models.RideRequest`. Optional `ride_type_id`, `promo_code`, `scheduled_at`. |
| GET | `/rides/:id` | Fetch one of your rides (rider or assigned driver). |
| GET | `/rides` | Paginated list (`page`, `per_page`) of rides for the authenticated rider/driver. |
| GET | `/rides/surge-info?latitude =..&longitude =..` | Returns current surge multiplier for the provided coordinates. |
| POST | `/rides/:id/cancel` | Cancels a ride. Accepts optional `reason` body. Riders can always cancel; drivers can cancel assigned rides. |
| POST | `/rides/:id/rate` | Submit a rating for a completed ride. Body matches `models.RideRatingRequest`. |

#### Driver endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/driver/rides/available` | List open ride requests that can be accepted. |
| POST | `/driver/rides/:id/accept` | Claim a requested ride. Fails if someone else already accepted. |
| POST | `/driver/rides/:id/start` | Move an accepted ride into `in_progress`. |
| POST | `/driver/rides/:id/complete` | Finalize the ride. Body: `{ "actual_distance": <km> }`. Computes fare adjustments and final status. |

#### Example: POST /api/v1/rides

```json
{
  "pickup_latitude": 40.758,
  "pickup_longitude": -73.9855,
  "pickup_address": "W 45th St, New York, NY",
  "dropoff_latitude": 40.7128,
  "dropoff_longitude": -74.006,
  "dropoff_address": "Financial District, NY",
  "ride_type_id": "2d6f...",
  "promo_code": "WELCOME20",
  "scheduled_at": null,
  "is_scheduled": false
}
```

Response (trimmed):

```json
{
  "success": true,
  "data": {
    "id": "c2e8...",
    "status": "requested",
    "estimated_distance": 8.3,
    "estimated_duration": 24,
    "estimated_fare": 23.5,
    "surge_multiplier": 1.15,
    "requested_at": "2025-01-09T15:04:05Z"
  }
}
```

#### Example: POST /api/v1/rides/:id/rate

```json
{
  "rating": 5,
  "feedback": "Great driver, clean car"
}
```

#### Example: POST /api/v1/driver/rides/:id/complete

```json
{
  "actual_distance": 9.1
}
```

Returns the updated ride, including `final_fare`, `actual_duration` and `completed_at` if the state transition succeeds.

#### Surge info response

`GET /api/v1/rides/surge-info?latitude =40.75&longitude =-73.98`

```json
{
  "success": true,
  "data": {
    "surge_multiplier": 1.3,
    "is_surge_active": true,
    "message": "Time-based surge pricing active"
  }
}
```
### Mobile API (:8087)

A convenience façade that reuses the rides and favorites handlers for mobile clients. Every route requires a JWT and lives under `http://localhost:8087/api/v1`.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/rides/history` | Offset-based history for the authenticated rider/driver. Query params: `status`, `start_date`, `end_date`, `limit` (default 20), `offset`. |
| GET | `/rides/:id/receipt` | Returns a receipt for completed rides owned by the caller. Includes fare breakdown & payment method. |
| POST | `/rides/:id/rate` | Same payload as the rides service; exposed here for mobile clients. |
| GET | `/profile` | Fetches rider/driver profile information via `service.GetUserProfile`. |
| PUT | `/profile` | Updates `first_name`, `last_name`, `phone_number`. |
| POST | `/favorites` | Create a favorite location. Body: `{ "name", "address", "latitude", "longitude" }`. |
| GET | `/favorites` | List favorite locations for the caller. |
| GET | `/favorites/:id` | Retrieve one favorite if it belongs to the caller. |
| PUT | `/favorites/:id` | Update a favorite location. Same schema as create. |
| DELETE | `/favorites/:id` | Delete a favorite location owned by the caller. |

#### Example: GET /api/v1/rides/history

```
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8087/api/v1/rides/history?limit=10&offset=0&status=completed&start_date=2025-01-01&end_date=2025-01-31"
```

Response:

```json
{
  "rides": [
    {
      "id": "c2e8...",
      "status": "completed",
      "pickup_address": "Midtown",
      "dropoff_address": "FiDi",
      "final_fare": 24.15,
      "completed_at": "2025-01-05T18:12:00Z"
    }
  ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

#### Example: GET /api/v1/rides/:id/receipt

```json
{
  "success": true,
  "data": {
    "ride_id": "c2e8...",
    "pickup_address": "Midtown",
    "dropoff_address": "FiDi",
    "distance": 8.9,
    "duration": 22,
    "base_fare": 19.5,
    "surge_multiplier": 1.2,
    "final_fare": 23.4,
    "payment_method": "wallet"
  }
}
```

#### Example: POST /api/v1/favorites

```json
{
  "name": "Home",
  "address": "123 Main St, Brooklyn, NY",
  "latitude": 40.6782,
  "longitude": -73.9442
}
```

The response is the stored `FavoriteLocation` struct with `id`, timestamps, and the caller's `user_id`.
### Geo Service (:8083)

Tracks driver coordinates in Redis and provides helper utilities.

- **Base URL:** `http://localhost:8083/api/v1/geo`
- **Auth:** All routes require a Bearer token. Driver updates additionally require the `driver` role.

| Method | Path | Description |
| --- | --- | --- |
| POST | `/location` | Drivers send `{ "latitude": <float>, "longitude": <float> }` to update their last known position. |
| GET | `/drivers/:id/location` | Look up a driver's last stored location by UUID. Returns `{ "latitude", "longitude", "updated_at" }`. |
| POST | `/distance` | Utility endpoint that accepts `{ "from_latitude", "from_longitude", "to_latitude", "to_longitude" }` and responds with `distance_km` + `eta_minutes`. |

#### Example: POST /api/v1/geo/distance

```json
{
  "from_latitude": 40.758,
  "from_longitude": -73.9855,
  "to_latitude": 40.7128,
  "to_longitude": -74.006
}
```

Response:

```json
{
  "success": true,
  "data": {
    "distance_km": 8.3,
    "eta_minutes": 22
  }
}
```
### Payments Service (:8084)

Wallet management, ride charge capture, and refund flows. Implemented in `internal/payments/handler.go`.

- **Base URL:** `http://localhost:8084/api/v1`
- **Auth:** All routes require JWT except the Stripe webhook.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/wallet` | Returns the caller's wallet (`models.Wallet`). Creates one on demand if needed. |
| POST | `/wallet/topup` | Body `{ "amount": <float>, "stripe_payment_method": "pm_xxx" }`. Charges Stripe when the key is configured, otherwise simulates success. |
| GET | `/wallet/transactions` | Query `limit`/`offset`. Returns an array of `models.WalletTransaction` + `meta`. |
| POST | `/payments/process` | Charges a ride. Request: `{ "ride_id": "uuid", "amount": 23.5, "payment_method": "wallet|stripe" }`. |
| GET | `/payments/:id` | Returns the payment if the caller is the rider or driver on the record. |
| POST | `/payments/:id/refund` | Admins and riders can request refunds. Body `{ "reason": "Driver never showed" }`. |
| POST | `/webhooks/stripe` | Unauthenticated endpoint for Stripe events. Payload must include `type` and `data.object.id`. |

#### Example: POST /api/v1/payments/process

```json
{
  "ride_id": "c2e8eb07-...",
  "amount": 23.50,
  "payment_method": "wallet"
}
```

On success the response contains the stored `models.Payment` with commission and driver earnings calculated.

#### Example: POST /api/v1/payments/:id/refund

```json
{
  "reason": "Driver cancelled mid ride"
}
```

Admins can refund any payment; riders can only refund their own. Refunds set the payment status to `refunded` and trigger Stripe refund logic when available.

#### Stripe webhook shape

```json
{
  "type": "payment_intent.succeeded",
  "data": {
    "object": {
      "id": "pi_123",
      "metadata": {
        "ride_id": "c2e8..."
      }
    }
  }
}
```

The handler currently verifies the payload format only; signature verification should be added before production use.
### Promos Service (:8089)

Promo codes, referral bonuses, ride types, and fare estimation helpers.

- **Base URL:** `http://localhost:8089/api/v1`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/ride-types` | None | Lists configured ride types (Economy, Premium, XL, etc.). |
| POST | `/ride-types/calculate-fare` | None | Body `{ "ride_type_id", "distance", "duration", "surge_multiplier" }`. Returns `fare`. |
| POST | `/promo-codes/validate` | Bearer | Validates `code` for the caller and ride amount. `{ "code": "WELCOME20", "ride_amount": 25 }`. |
| GET | `/referrals/my-code` | Bearer | Generates/returns the caller's referral code. |
| POST | `/referrals/apply` | Bearer | Body `{ "referral_code": "RILEY25" }`. Applies referral bonuses. |
| POST | `/admin/promo-codes` | Admin | Creates a promo code. Payload mirrors `internal/promos.PromoCode`. |

#### Example: POST /api/v1/promo-codes/validate

```json
{
  "code": "WELCOME20",
  "ride_amount": 30.0
}
```

Response:

```json
{
  "success": true,
  "data": {
    "valid": true,
    "message": "Promo applied",
    "discount_amount": 6.0,
    "final_amount": 24.0
  }
}
```

#### Example: POST /api/v1/ride-types/calculate-fare

```json
{
  "ride_type_id": "2d6f...",
  "distance": 12.3,
  "duration": 28,
  "surge_multiplier": 1.2
}
```

Response contains the calculated fare and echoes the request metadata.

#### Example: POST /api/v1/admin/promo-codes

```json
{
  "code": "SUMMER25",
  "description": "25% off up to $10",
  "discount_type": "percentage",
  "discount_value": 0.25,
  "max_discount_amount": 10,
  "min_ride_amount": 12,
  "uses_per_user": 1,
  "valid_from": "2025-06-01T00:00:00Z",
  "valid_until": "2025-08-31T23:59:59Z"
}
```

The handler injects `created_by` based on the authenticated admin and persists the promo code.
### Notifications Service (:8085)

Multi-channel messaging (Firebase push, Twilio SMS, SMTP email) plus ride lifecycle notifications.

- **Base URL:** `http://localhost:8085/api/v1`
- **Auth:** Every route uses the shared JWT middleware; ride lifecycle + admin endpoints should be called by trusted services using admin tokens.

#### User-facing endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/notifications` | List notifications for the caller. Query `limit`/`offset`. Envelope includes `meta`. |
| GET | `/notifications/unread/count` | Returns `{ "count": <int> }` with the unread total. |
| POST | `/notifications/:id/read` | Marks a notification as read. No body required. |
| POST | `/notifications/send` | Send a single notification. Body below. |
| POST | `/notifications/schedule` | Schedule a notification for the future. Requires `scheduled_at` RFC3339. |

`SendNotification` / `ScheduleNotification` payload:

```json
{
  "user_id": "5a4e...",
  "type": "ride_completed",
  "channel": "push",
  "title": "Your ride is done",
  "body": "Thanks for riding with us!",
  "data": {
    "ride_id": "c2e8..."
  },
  "scheduled_at": "2025-01-09T18:00:00Z" // only for schedule
}
```

#### Ride lifecycle hooks

| Method | Path | Description |
| --- | --- | --- |
| POST | `/notifications/ride/requested` |
| POST | `/notifications/ride/accepted` |
| POST | `/notifications/ride/started` |
| POST | `/notifications/ride/completed` |
| POST | `/notifications/ride/cancelled` |

These endpoints accept variants of `RideNotificationRequest`:

```json
{
  "user_id": "5a4e...",
  "ride_id": "c2e8...",
  "data": {
    "pickup": "Midtown",
    "driver_name": "Sofia"
  }
}
```

`/ride/cancelled` additionally requires `cancelled_by` ("driver" or "rider"). Use service credentials so the middleware allows the call.

#### Admin bulk broadcast

| Method | Path | Description |
| --- | --- | --- |
| POST | `/admin/notifications/bulk` | Admin-only. Body `{ "user_ids": ["..."], "type": "promo", "channel": "email", "title": "...", "body": "...", "data": {}}`. Returns how many notifications were queued. |
### Real-time Service (:8086)

Provides WebSocket connectivity plus helper REST endpoints for chat history and broadcasting updates. See `internal/realtime/handler.go`.

- **Base URL:** `http://localhost:8086/api/v1`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/ws` | Bearer | Upgrades to a WebSocket connection. JWT determines the user ID + role tracked inside the hub. |
| GET | `/rides/:ride_id/chat` | Bearer | Returns `{ "ride_id", "messages": [...] }` for rides the caller belongs to. |
| GET | `/drivers/:driver_id/location` | Bearer | Fetches the latest driver coordinates stored in Redis. |
| GET | `/stats` | Admin | Connection counts, hub stats. |
| POST | `/internal/broadcast/ride` | Network-restricted | Body `{ "ride_id": "...", "data": { ... } }`, pushes an event to everyone watching the ride. |
| POST | `/internal/broadcast/user` | Network-restricted | Body `{ "user_id": "...", "type": "notification", "data": { ... } }`. |

#### WebSocket handshake example

```js
const socket = new WebSocket("ws://localhost:8086/api/v1/ws", {
  headers: { Authorization: `Bearer ${token}` }
});
```

After connecting, the client receives events broadcast by other services (ride status, chat messages, etc.) and can send structured JSON payloads per the `pkg/websocket` client contract.

> **Security note:** The `/internal/broadcast/*` routes do not attach middleware today. Deploy them behind mTLS/network ACLs or add auth middleware before exposing them in production.
### Admin Service (:8088)

Back-office operations accessible only to admin accounts. Routes live under `http://localhost:8088/api/v1/admin` and every request passes through both `AuthMiddleware` and `RequireAdmin()`.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/dashboard` | Returns aggregated `DashboardStats` (user totals + ride revenue snapshots). |
| GET | `/users` | Paginated users list. Query `limit` (default 20, max 100) & `offset`. |
| GET | `/users/:id` | Fetch one user. |
| POST | `/users/:id/suspend` | Suspends the account (no body needed). |
| POST | `/users/:id/activate` | Re-activates an account. |
| GET | `/drivers/pending` | Drivers awaiting manual approval. |
| POST | `/drivers/:id/approve` | Marks the driver as approved. |
| POST | `/drivers/:id/reject` | Rejects the pending driver. |
| GET | `/rides/recent?limit=50` | Latest rides (default 50, cap 100). Helpful for monitoring. |
| GET | `/rides/stats?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD` | Returns `RideStats` (total, completed, cancelled, revenue, avg fare). |

Sample `GET /api/v1/admin/dashboard` response:

```json
{
  "success": true,
  "data": {
    "users": {
      "total_users": 120000,
      "total_riders": 95000,
      "total_drivers": 25000,
      "active_users": 38000
    },
    "rides": {
      "total_rides": 210000,
      "completed_rides": 180000,
      "cancelled_rides": 15000,
      "total_revenue": 4200000,
      "avg_fare": 23.3
    },
    "today_rides": {
      "total_rides": 3200,
      "completed_rides": 2950,
      "cancelled_rides": 180,
      "total_revenue": 72000,
      "avg_fare": 22.5
    }
  }
}
```
### Analytics Service (:8091)

Business intelligence endpoints for admins. Middleware enforces both JWT + admin role.

- **Base URL:** `http://localhost:8091/api/v1/analytics`
- **Date handling:** `start_date`/`end_date` default to the last 30 days when omitted. Pass `YYYY-MM-DD`.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/dashboard` | Lightweight snapshot (active rides, revenue today, active users, etc.). |
| GET | `/revenue` | Query `start_date`, `end_date`. Returns `RevenueMetrics`. |
| GET | `/promo-codes` | Promo performance per code. Same date params. |
| GET | `/ride-types` | Usage mix across ride types. |
| GET | `/referrals` | Referral funnel metrics, conversion rate and bonus spend. |
| GET | `/top-drivers?limit=10` | Top performing drivers within the window (limit 1-100). |
| GET | `/heat-map?grid_size=0.01` | Geographic demand data suitable for heat maps. `grid_size` is in degrees (~0.01 ≈ 1 km). |
| GET | `/financial-report` | Profit & loss style report for the period. |
| GET | `/demand-zones?min_rides=20` | Highlights zones exceeding the provided ride count. |

Example revenue response:

```json
{
  "success": true,
  "data": {
    "period": "2025-01-01/2025-01-31",
    "total_revenue": 4200000,
    "total_rides": 185000,
    "avg_fare_per_ride": 22.7,
    "total_discounts": 380000,
    "platform_earnings": 950000,
    "driver_earnings": 3250000
  }
}
```
### Fraud Service (:8092)

Admin-only APIs for fraud detection, triage, and enforcement. All routes require the admin role and live under `http://localhost:8092/api/v1/fraud`.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/alerts` | Paginated (`page`, `per_page`) list of pending alerts. |
| GET | `/alerts/:id` | Fetch a single alert. |
| POST | `/alerts` | Manually create an alert. See payload below. |
| PUT | `/alerts/:id/investigate` | Body `{ "notes": "Investigating payment anomalies" }`. Sets status to `investigating` and logs admin ID. |
| PUT | `/alerts/:id/resolve` | Body `{ "confirmed": true, "notes": "Chargebacks confirmed", "action_taken": "suspended" }`. |
| GET | `/users/:id/alerts` | Alerts for one user (`page`, `per_page`). |
| GET | `/users/:id/risk-profile` | Returns `UserRiskProfile`. |
| POST | `/users/:id/analyze` | Runs the full analysis pipeline and returns the latest profile snapshot. |
| POST | `/users/:id/suspend` | Body `{ "reason": "Confirmed payment fraud" }`. Suspends via the fraud service. |
| POST | `/users/:id/reinstate` | Body `{ "reason": "False positive" }`. Reinstates account. |
| POST | `/detect/payment/:user_id` | Triggers automated payment fraud checks. No body. |
| POST | `/detect/ride/:user_id` | Triggers ride pattern checks. No body. |

`POST /alerts` payload (matches `CreateAlertRequest`):

```json
{
  "user_id": "5a4e...",
  "alert_type": "payment_fraud",
  "alert_level": "high",
  "description": "Multiple failed cards followed by success",
  "details": {
    "failed_attempts": 5,
    "last_payment_id": "pay_123"
  },
  "risk_score": 87.5
}
```

Alert types: `payment_fraud`, `account_fraud`, `location_fraud`, `ride_fraud`, `rating_manipulation`, `promo_abuse`. Alert levels: `low`, `medium`, `high`, `critical`.
### ML ETA Service (:8093)

Machine-learning driven ETA predictions plus model management endpoints.

- **Base URL:** `http://localhost:8093/api/v1/eta`
- **Auth:** `POST /predict` and `/predict/batch` are public. Everything else requires JWT; admin-only routes additionally enforce `RequireAdmin()`.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/predict` | None | Body matches `ETAPredictionRequest` (coords, traffic level, weather, driver_id, ride_type_id). |
| POST | `/predict/batch` | None | `{ "routes": [ ETAPredictionRequest, ... ] }` (max 100). |
| POST | `/train` | Admin | Starts asynchronous model retraining. Returns `202 Accepted`. |
| GET | `/model/stats` | Admin | Summary (version, training samples, accuracy, last_trained_at). |
| GET | `/model/accuracy?days=30` | Admin | Aggregated accuracy metrics for the requested window (1-365 days). |
| POST | `/model/tune` | Admin | Adjust hyper-parameters. Accepts any subset of the weights (`distance_weight`, `traffic_weight`, etc.) as floats 0-1. |
| GET | `/analytics/predictions?limit=50&offset=0` | Bearer | Historical prediction rows. |
| GET | `/analytics/accuracy?days=30` | Bearer | Accuracy trend data. |
| GET | `/analytics/features` | Bearer | Feature importance (distance vs traffic vs weather, etc.). |

#### Example: POST /api/v1/eta/predict

```json
{
  "pickup_latitude": 40.758,
  "pickup_longitude": -73.9855,
  "dropoff_latitude": 40.7128,
  "dropoff_longitude": -74.006,
  "traffic_level": "high",
  "weather": "rain",
  "driver_id": "5a4e...",
  "ride_type_id": 1
}
```

Response:

```json
{
  "success": true,
  "data": {
    "estimated_minutes": 22.4,
    "estimated_seconds": 1344,
    "distance_km": 8.3,
    "confidence": 0.82,
    "model_version": "v1.0-ml",
    "predicted_arrival_time": "2025-01-09T15:34:00Z",
    "factors": {
      "base_eta": 19.6,
      "traffic": 1.3,
      "time_of_day": 1.1,
      "weather": 1.15,
      "historical_eta": 21.8
    }
  }
}
```
### Scheduler Service (:8090)

The scheduler is a background worker that picks up scheduled rides and time-based tasks (see `internal/scheduler/worker.go`). It does **not** expose application APIs—only the standard diagnostics endpoints:

- `GET /healthz`
- `GET /version`
- `GET /metrics`

If you need to enqueue new scheduled rides or notifications, call the relevant ride/notification services; the scheduler polls the database and notifications service URL configured through `NOTIFICATIONS_SERVICE_URL`.
### Health, metrics & observability

Every HTTP service (including scheduler) exposes the same trio of operational endpoints:

| Path | Description |
| --- | --- |
| `GET /healthz` | Returns `{ "service": "<name>", "status": "healthy" }` via `pkg/common.HealthCheck`. Use for readiness/liveness probes. |
| `GET /version` | Where defined, returns the service name + semantic version string declared in `cmd/<service>/main.go`. |
| `GET /metrics` | Prometheus scrape endpoint (Gin wraps `promhttp.Handler`). |

All Gin routers also install middleware for structured logging (`pkg/logger` + `middleware.RequestLogger`), correlation IDs, CORS, security headers, and request sanitisation.

If you deploy behind Kong or Istio, surface only the `/api/v1/...` routes externally and keep `/metrics` on the internal mesh/Grafana scrape path.
