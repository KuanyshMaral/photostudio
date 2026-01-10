# API Testing Script

Этот документ содержит команды для тестирования всех endpoints API через curl.

## Подготовка

```bash
# 1. Запустить приложение
make dev

# 2. В отдельном терминале заполнить БД тестовыми данными
make seed
```

## Переменные для удобства

```bash
# Базовый URL
BASE_URL="http://localhost:3001/api/v1"

# Токены будут получены после входа
ADMIN_TOKEN=""
OWNER_TOKEN=""
CLIENT_TOKEN=""
```

## 1. Authentication Endpoints

### 1.1 Register Client

```bash
curl -X POST "$BASE_URL/auth/register/client" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newclient@test.com",
    "password": "Password123!",
    "name": "New Test Client",
    "phone": "+7 777 999 9999"
  }'
```

**Expected Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "user": {...},
    "token": "eyJhbGc..."
  }
}
```

### 1.2 Register Studio Owner

```bash
curl -X POST "$BASE_URL/auth/register/studio" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newowner@test.com",
    "password": "Password123!",
    "name": "New Studio Owner",
    "phone": "+7 777 888 8888",
    "company_name": "New Studio LLC",
    "bin": "987654321098"
  }'
```

**Expected Response:** `201 Created`

### 1.3 Login

```bash
# Login as client
curl -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "client1@test.com",
    "password": "client123"
  }'

# Save token
CLIENT_TOKEN="<token_from_response>"

# Login as owner
curl -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "owner1@studio.kz",
    "password": "owner123"
  }'

# Save token
OWNER_TOKEN="<token_from_response>"

# Login as admin
curl -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@studiobooking.kz",
    "password": "admin123"
  }'

# Save token
ADMIN_TOKEN="<token_from_response>"
```

**Expected Response:** `200 OK`

### 1.4 Get Profile

```bash
curl -X GET "$BASE_URL/auth/me" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

**Expected Response:** `200 OK`

### 1.5 Update Profile

```bash
curl -X PUT "$BASE_URL/auth/me" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Client Name",
    "phone": "+7 777 111 2222"
  }'
```

**Expected Response:** `200 OK`

## 2. Catalog Endpoints

### 2.1 Get All Studios

```bash
curl -X GET "$BASE_URL/studios"
```

**Expected Response:** `200 OK` with array of studios

### 2.2 Get Studios with Filters

```bash
# Filter by city
curl -X GET "$BASE_URL/studios?city=Алматы"

# Filter by district
curl -X GET "$BASE_URL/studios?district=Алмалинский"

# Filter by rating
curl -X GET "$BASE_URL/studios?min_rating=4.5"

# Combined filters
curl -X GET "$BASE_URL/studios?city=Алматы&min_rating=4.5&limit=5"
```

**Expected Response:** `200 OK`

### 2.3 Get Studio by ID

```bash
# Replace {studio_id} with actual ID (e.g., 1)
curl -X GET "$BASE_URL/studios/1"
```

**Expected Response:** `200 OK` with studio details

### 2.4 Get Studio Rooms

```bash
curl -X GET "$BASE_URL/studios/1/rooms"
```

**Expected Response:** `200 OK` with array of rooms

### 2.5 Search Studios

```bash
curl -X GET "$BASE_URL/studios/search?q=Light"
```

**Expected Response:** `200 OK`

## 3. Studio Owner Endpoints

### 3.1 Create Studio (Verified Owner Only)

```bash
curl -X POST "$BASE_URL/studios" \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Test Studio",
    "description": "A brand new test studio",
    "address": "Test Street, 123",
    "district": "Алмалинский",
    "city": "Алматы",
    "phone": "+7 727 999 9999"
  }'
```

**Expected Response:** `201 Created`

### 3.2 Get My Studios

```bash
curl -X GET "$BASE_URL/studios/my" \
  -H "Authorization: Bearer $OWNER_TOKEN"
```

**Expected Response:** `200 OK`

### 3.3 Update Studio

```bash
# Replace {studio_id} with your studio ID
curl -X PUT "$BASE_URL/studios/1" \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Studio Name",
    "description": "Updated description"
  }'
```

**Expected Response:** `200 OK`

### 3.4 Create Room

```bash
curl -X POST "$BASE_URL/studios/1/rooms" \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Test Room",
    "description": "A test room",
    "capacity": 8,
    "area_sqm": 40,
    "room_type": "Portrait",
    "price_per_hour_min": 7000
  }'
```

**Expected Response:** `201 Created`

## 4. Booking Endpoints

### 4.1 Create Booking

```bash
# Get current time + 2 days for start_time
# Format: 2024-01-15T14:00:00Z

curl -X POST "$BASE_URL/bookings" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "start_time": "2026-01-13T14:00:00Z",
    "end_time": "2026-01-13T16:00:00Z"
  }'
```

**Expected Response:** `201 Created`

### 4.2 Get My Bookings

```bash
curl -X GET "$BASE_URL/bookings/my" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

**Expected Response:** `200 OK`

### 4.3 Get Booking Details

```bash
curl -X GET "$BASE_URL/bookings/1" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

**Expected Response:** `200 OK`

### 4.4 Update Booking Status (Owner)

```bash
curl -X PATCH "$BASE_URL/bookings/1/status" \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "confirmed"
  }'
```

**Expected Response:** `200 OK`

### 4.5 Cancel Booking (Client)

```bash
curl -X DELETE "$BASE_URL/bookings/1" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

**Expected Response:** `200 OK`

### 4.6 Get Studio Bookings (Owner)

```bash
curl -X GET "$BASE_URL/studios/1/bookings" \
  -H "Authorization: Bearer $OWNER_TOKEN"
```

**Expected Response:** `200 OK`

## 5. Review Endpoints

### 5.1 Create Review

```bash
# Note: Can only review completed bookings
curl -X POST "$BASE_URL/reviews" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "studio_id": 1,
    "booking_id": 1,
    "rating": 5,
    "comment": "Excellent studio! Highly recommend."
  }'
```

**Expected Response:** `201 Created`

### 5.2 Get Studio Reviews

```bash
curl -X GET "$BASE_URL/studios/1/reviews"
```

**Expected Response:** `200 OK`

### 5.3 Add Owner Response to Review

```bash
curl -X POST "$BASE_URL/reviews/1/response" \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "response": "Thank you for your feedback!"
  }'
```

**Expected Response:** `200 OK`

## 6. Notification Endpoints

### 6.1 Get My Notifications

```bash
curl -X GET "$BASE_URL/notifications" \
  -H "Authorization: Bearer $OWNER_TOKEN"
```

**Expected Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "notifications": [...],
    "unread_count": 5
  }
}
```

### 6.2 Mark Notification as Read

```bash
curl -X PATCH "$BASE_URL/notifications/1/read" \
  -H "Authorization: Bearer $OWNER_TOKEN"
```

**Expected Response:** `200 OK`

### 6.3 Mark All Notifications as Read

```bash
curl -X PATCH "$BASE_URL/notifications/read-all" \
  -H "Authorization: Bearer $OWNER_TOKEN"
```

**Expected Response:** `200 OK`

## 7. Admin Endpoints

### 7.1 Get Dashboard Stats

```bash
curl -X GET "$BASE_URL/admin/stats" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "total_users": 15,
    "total_studios": 5,
    "total_bookings": 50,
    "total_revenue": 1500000,
    "pending_verifications": 1
  }
}
```

### 7.2 Get Pending Verifications

```bash
curl -X GET "$BASE_URL/admin/verifications/pending" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`

### 7.3 Verify Studio Owner

```bash
curl -X POST "$BASE_URL/admin/verifications/1/approve" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "admin_notes": "All documents verified"
  }'
```

**Expected Response:** `200 OK`

### 7.4 Reject Studio Owner Verification

```bash
curl -X POST "$BASE_URL/admin/verifications/1/reject" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Incomplete documents",
    "admin_notes": "Missing BIN certificate"
  }'
```

**Expected Response:** `200 OK`

### 7.5 Get All Users

```bash
curl -X GET "$BASE_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`

### 7.6 Get All Studios

```bash
curl -X GET "$BASE_URL/admin/studios" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`

### 7.7 Get All Bookings

```bash
curl -X GET "$BASE_URL/admin/bookings" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`

### 7.8 Get All Reviews

```bash
curl -X GET "$BASE_URL/admin/reviews" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Expected Response:** `200 OK`

## Testing Checklist

### Authentication
- [Y ] Client registration works
- [Y ] Studio owner registration works
- [Y ] Login works for all roles
- [Y ] Get profile works
- [Y ] Update profile works
- [Y ] Invalid credentials are rejected
- [Y ] Weak passwords are rejected

### Catalog
- [Y ] Get all studios works
- [Y ] Filters work (city, room type, price)
- [Y ] Get studio by ID works
- [Y ] Get studio rooms works
- [Y ] Search works
- [Y ] Invalid studio ID returns 404

### Studio Management
- [Y ] Verified owner can create studio
- [Y ] Unverified owner cannot create studio
- [Y ] Get my studios works
- [Y ] Update studio works
- [Y ] Create room works
- [Y ] Only owner can modify their studios

### Booking
- [? ] Create booking works
- [? ] Get my bookings works
- [? ] Get booking details works
- [? ] Owner can update booking status
- [? ] Client can cancel booking
- [Y ] Get studio bookings works (owner only)
- [? ] Cannot book in the past
- [? ] Cannot double-book room

### Reviews
- [? ] Can create review for completed booking
- [Y ] Cannot review without booking
- [Y ] Get studio reviews works
- [Y ] Owner can respond to review
- [? ] Cannot review twice for same booking

### Notifications
- [Y ] Get notifications works
- [Y ] Mark as read works
- [Y ] Mark all as read works
- [Y ] Unread count is accurate
- [Y ] Only user's notifications are visible

### Admin
- [Y ] Dashboard stats work
- [Y ] Get pending verifications works
- [Y ] Approve verification works
- [Y ] Reject verification works
- [Y ] Get all users works
- [Y ] Get all studios works
- [Y ] Get all bookings works
- [Y ] Get all reviews works
- [Y ] Only admin can access admin endpoints

## Error Cases to Test

```bash
# 1. Unauthorized access
curl -X GET "$BASE_URL/bookings/my"
# Expected: 401 Unauthorized

# 2. Invalid token
curl -X GET "$BASE_URL/bookings/my" \
  -H "Authorization: Bearer invalid_token"
# Expected: 401 Unauthorized

# 3. Non-existent resource
curl -X GET "$BASE_URL/studios/999999"
# Expected: 404 Not Found

# 4. Invalid data
curl -X POST "$BASE_URL/auth/register/client" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "invalid-email",
    "password": "weak"
  }'
# Expected: 400 Bad Request

# 5. Duplicate email
curl -X POST "$BASE_URL/auth/register/client" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "client1@test.com",
    "password": "Password123!"
  }'
# Expected: 409 Conflict

# 6. Unauthorized action
curl -X POST "$BASE_URL/studios" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test"
  }'
# Expected: 403 Forbidden
```

## Notes

1. Replace `{id}` placeholders with actual IDs from your database
2. Adjust timestamps for bookings to be in the future
3. Some endpoints require specific user roles
4. Studio owners must be verified before creating studios
5. Reviews can only be created for completed bookings

## Automated Testing

Run the E2E test suite:

```bash
make e2e
```

This will test all endpoints automatically with proper setup and teardown.
