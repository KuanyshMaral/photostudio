# MWork Integration - Quick Start Guide

## 🎯 What Was Implemented

### Problem
MWork uses UUID for user IDs, but PhotoStudio uses int64 auto-increment IDs. When MWork tries to create a booking, it can't provide the correct PhotoStudio user ID.

### Solution
**Middleware-based ID mapping**: PhotoStudio middleware automatically maps `X-MWork-User-ID` (UUID) → PhotoStudio internal ID (int64).

---

## 📁 Files Created/Modified

### PhotoStudio Backend

1. **New Middleware**: `internal/middleware/mwork_user.go`
   - Function: `MWorkUserAuth(userRepo)` 
   - Maps MWork UUID to PhotoStudio int64 ID
   - Sets `user_id` and `role` in Gin context

2. **Updated Routes**: `cmd/api/main.go`
   - Added `/internal/mwork/*` endpoints
   - Protected with `InternalTokenAuth()` + `MWorkUserAuth()`

3. **Documentation**: 
   - `docs/MWORK_BOOKING_INTEGRATION.md` - Full integration guide
   - `docs/MWORK_INTEGRATION_QUICKSTART.md` - This file

4. **Tests**:
   - `tests/test_mwork_booking.sh` - Bash test script
   - `tests/test_mwork_booking.ps1` - PowerShell test script

---

## 🚀 Quick Start

### 1. Configure PhotoStudio (.env)

```env
PORT=8090
DATABASE_URL=postgresql://user:pass@localhost:5432/photostudio

# MWork Integration
MWORK_SYNC_ENABLED=true
MWORK_SYNC_TOKEN=my-secret-token-123
MWORK_SYNC_ALLOWED_IPS=127.0.0.1,localhost
```

### 2. Start PhotoStudio

```bash
cd photostudio-main
go run cmd/api/main.go
```

### 3. Test with cURL

```bash
# Replace with actual MWork User UUID from database
MWORK_USER_ID="550e8400-e29b-41d4-a716-446655440000"

# Create booking
curl -X POST http://localhost:8090/internal/mwork/bookings \
  -H "Authorization: Bearer my-secret-token-123" \
  -H "X-MWork-User-ID: $MWORK_USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "studio_id": 1,
    "start_time": "2026-02-15T10:00:00Z",
    "end_time": "2026-02-15T12:00:00Z"
  }'
```

---

## 🔑 Required Headers

Every request from MWork must include:

```http
Authorization: Bearer <MWORK_SYNC_TOKEN>
X-MWork-User-ID: <uuid-of-mwork-user>
Content-Type: application/json
```

---

## 📍 Available Endpoints

All endpoints under `/internal/mwork/*`:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/bookings` | Create booking |
| GET | `/bookings` | List user's bookings |
| GET | `/studios` | List studios with filters |
| GET | `/studios/:id` | Get studio details |
| GET | `/rooms/:id/availability` | Check room availability |
| GET | `/rooms/:id/busy-slots` | Get busy time slots |

---

## 🧪 Testing

### PowerShell (Windows)
```powershell
cd photostudio-main\tests
.\test_mwork_booking.ps1
```

### Bash (Linux/Mac)
```bash
cd photostudio-main/tests
chmod +x test_mwork_booking.sh
./test_mwork_booking.sh
```

---

## 🐛 Common Errors

### Error: "USER_NOT_SYNCED"
```json
{
  "error": {
    "code": "USER_NOT_SYNCED",
    "message": "User not found in PhotoStudio..."
  }
}
```
**Cause**: User was never synced or sync failed  
**Solution**: Trigger manual sync via `/internal/mwork/users/sync`

### Error: "AUTH_INVALID"
```json
{
  "error": {
    "code": "AUTH_INVALID",
    "message": "Invalid or missing internal token"
  }
}
```
**Cause**: Wrong MWORK_SYNC_TOKEN  
**Solution**: Check token in PhotoStudio .env

### Error: "MWORK_USER_ID_MISSING"
```json
{
  "error": {
    "code": "MWORK_USER_ID_MISSING",
    "message": "X-MWork-User-ID header is required"
  }
}
```
**Cause**: Missing header  
**Solution**: Add `X-MWork-User-ID` header to request

---

## 🔍 How It Works (Step by Step)

```
1. MWork User logs in
   ├─ Gets JWT with user.ID (UUID)
   └─ Stores UUID: 550e8400-e29b-41d4-a716-446655440000

2. MWork sends booking request
   ├─ POST /internal/mwork/bookings
   ├─ Header: Authorization: Bearer <token>
   ├─ Header: X-MWork-User-ID: 550e8400-...
   └─ Body: { room_id: 1, studio_id: 1, ... }

3. PhotoStudio middleware intercepts
   ├─ Validates MWORK_SYNC_TOKEN
   ├─ Extracts X-MWork-User-ID header
   ├─ Queries DB: SELECT * FROM users WHERE mwork_user_id = '550e8400-...'
   ├─ Found: { id: 42, email: "user@example.com", role: "client" }
   └─ Sets context: c.Set("user_id", 42)

4. BookingHandler receives request
   ├─ Reads: userID := c.GetInt64("user_id") // 42
   └─ Creates booking with PhotoStudio ID

5. Response sent back to MWork
   └─ { booking_id: 123, status: "pending", ... }
```

---

## 📊 Database Schema

### PhotoStudio users table
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,                    -- PhotoStudio internal ID
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL,
    
    mwork_user_id UUID NULL,                  -- Link to MWork
    mwork_role TEXT NULL,                     -- MWork role (for audit)
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_mwork_user_id_unique
    ON users(mwork_user_id)
    WHERE mwork_user_id IS NOT NULL;
```

### MWork users table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,                      -- MWork user ID
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Mapping
```
MWork.users.id (UUID) ──────> PhotoStudio.users.mwork_user_id (UUID)
                                      │
                                      └──> PhotoStudio.users.id (int64)
```

---

## ✅ Checklist for Production

- [ ] Set strong `MWORK_SYNC_TOKEN` in both services
- [ ] Configure `MWORK_SYNC_ALLOWED_IPS` for security
- [ ] Test all endpoints with real MWork users
- [ ] Monitor sync failures (implement retry mechanism)
- [ ] Add logging for audit trail
- [ ] Set up alerts for "USER_NOT_SYNCED" errors
- [ ] Document internal API in Swagger/OpenAPI
- [ ] Load test with concurrent requests

---

## 📞 Support

For issues or questions:
1. Check error codes above
2. Review logs in PhotoStudio
3. Verify user exists in PhotoStudio DB:
   ```sql
   SELECT * FROM users WHERE mwork_user_id = '<uuid>';
   ```
4. Trigger manual sync if needed

---

## 🎓 Next Steps

1. **Implement in MWork**: Copy client code from `MWORK_BOOKING_INTEGRATION.md`
2. **Add Retry Logic**: Handle sync failures gracefully
3. **Monitoring**: Track sync success/failure rates
4. **Documentation**: Update API docs with new endpoints

---

**Last Updated**: 2026-02-09  
**Version**: 1.0.0
