# API Testing Results - Day 1

## Test Environment
- Server: http://localhost:3001
- Database: PostgreSQL / SQLite
- Date: Day 1

## Auth Module Tests
| Endpoint                     | Method | Status    | Notes                                 |
|------------------------------|--------|-----------|---------------------------------------|
| /api/v1/auth/register/client | POST   | ⬜ Pending | Register new client user              |
| /api/v1/auth/register/studio | POST   | ⬜ Pending | Register new studio owner             |
| /api/v1/auth/login           | POST   | ⬜ Pending | Login and get JWT token               |
| /api/v1/users/me             | GET    | ⬜ Pending | Get current user info (requires auth) |
| /api/v1/users/me             | PUT    | ⬜ Pending | Update user profile (requires auth)   |

## Catalog Module Tests
| Endpoint            | Method | Status    | Notes                     |
|---------------------|--------|-----------|---------------------------|
| /api/v1/studios     | GET    | ⬜ Pending | List all verified studios |
| /api/v1/studios/:id | GET    | ⬜ Pending | Get studio details        |
| /api/v1/rooms       | GET    | ⬜ Pending | List rooms with filters   |
| /api/v1/rooms/:id   | GET    | ⬜ Pending | Get room details          |

## Booking Module Tests
| Endpoint                       | Method | Status    | Notes                               |
|--------------------------------|--------|-----------|-------------------------------------|
| /api/v1/bookings               | POST   | ⬜ Pending | Create new booking (requires auth)  |
| /api/v1/users/me/bookings      | GET    | ⬜ Pending | Get user's bookings (requires auth) |
| /api/v1/rooms/:id/availability | GET    | ⬜ Pending | Check room availability for date    |

## Review Module Tests
| Endpoint                    | Method | Status    | Notes                         |
|-----------------------------|--------|-----------|-------------------------------|
| /api/v1/reviews             | POST   | ⬜ Pending | Create review (requires auth) |
| /api/v1/studios/:id/reviews | GET    | ⬜ Pending | Get studio reviews            |

## Issues Found
Issue 1: Middleware/Handler Mismatch in CreateStudio

Status: 🔧 Fixed
Severity: High
Endpoint: POST /api/v1/studios
Description: CreateStudio handler was looking for user object in context, but JWT middleware only sets user_id and role
Symptom: Always returned 401 "User not authenticated" even with valid token
Root Cause:

Middleware (auth.go line 159-161) stores: c.Set("user_id", claims.UserID) and c.Set("role", claims.Role)
Handler (catalog/handler.go line 137) expected: user, exists := c.Get("user")
Result: exists was always false because key name mismatch


Fix Applied: Changed CreateStudio handler to use c.GetInt64("user_id") instead of c.Get("user")
File Changed: internal/modules/catalog/handler.go lines 136-151
Verified: UpdateStudio and CreateRoom handlers already use correct approach

Issue 2: Test Date Validation Failure

Status: 🔧 Fixed
Severity: Medium
Endpoint: N/A (Unit tests)
Description: Unit tests used past dates (2025-12-31) which failed business logic validation
Symptom: Tests failed with "validation error" - start time cannot be in the past
Root Cause: Service validates req.StartTime.Before(now) and rejects bookings in the past
Fix Applied: Changed all test dates from 2025 to 2026
Files Changed: internal/modules/booking/service_test.go (lines 84, 144, 152, 172, 173, 197, 240)

Issue 3: Test Day-of-Week Mismatch

Status: 🔧 Fixed
Severity: Medium
Endpoint: N/A (Unit tests)
Description: Tests expected Wednesday but used Thursday date (2026-12-31)
Symptom: GetRoomAvailability tests returned empty slots array (expected 2 slots)
Root Cause:

Tests mocked working hours for "wednesday"
Used date 2026-12-31 which is Thursday
Service couldn't find working hours for Thursday, returned no available slots


Fix Applied: Changed dates in tests 3 and 6 from 2026-12-31 to 2026-12-30 (Wednesday)
Files Changed: internal/modules/booking/service_test.go (lines 144, 152, 240)

Issue 4: PostgreSQL Migration Type Casting Error

Status: ⚠️ Documented (Not fixed - design decision)
Severity: Low
Endpoint: N/A (Database migration)
Description: GORM AutoMigrate cannot automatically convert existing PostgreSQL columns from text[] to jsonb
Error Message: ОШИБКА: привести тип text[] к jsonb нельзя (SQLSTATE 42846)
Affected: Domain.Room model - amenities field
Workaround: Use SQLite for development, or drop and recreate PostgreSQL database
Recommendation: Use explicit migration files instead of AutoMigrate for production
## Test Checklist
- [ ] All endpoints documented
- [ ] Test credentials prepared
- [ ] Postman/curl commands ready
- [ ] Response validation defined
- [ ] Error cases tested
