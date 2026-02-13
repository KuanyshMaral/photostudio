# MWork → PhotoStudio Booking Integration Guide

## Overview

This guide explains how to integrate PhotoStudio booking functionality into MWork backend using the internal API with automatic user ID mapping.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MWork Backend                            │
│                                                             │
│  User Login → JWT with user.ID (UUID)                      │
│       ↓                                                     │
│  Add headers:                                               │
│    Authorization: Bearer <MWORK_SYNC_TOKEN>                │
│    X-MWork-User-ID: <user.ID>                              │
│       ↓                                                     │
│  HTTP Client → POST /internal/mwork/bookings               │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           │  HTTP Request
                           ↓
┌─────────────────────────────────────────────────────────────┐
│              PhotoStudio Backend (Port 8090)                │
│                                                             │
│  Middleware: MWorkUserAuth                                  │
│    1. Validate MWORK_SYNC_TOKEN                            │
│    2. Extract X-MWork-User-ID header                       │
│    3. DB Lookup: users WHERE mwork_user_id = ?             │
│    4. c.Set("user_id", foundUser.ID) ← int64              │
│       ↓                                                     │
│  BookingHandler.CreateBooking                              │
│    - Uses c.GetInt64("user_id") from context              │
│    - Creates booking with PhotoStudio internal ID          │
└─────────────────────────────────────────────────────────────┘
```

---

## API Endpoints

### Base URL
```
http://localhost:8090/internal/mwork
```

### Authentication
All requests require:
```http
Authorization: Bearer <MWORK_SYNC_TOKEN>
X-MWork-User-ID: <uuid-from-mwork>
```

### Available Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/bookings` | Create a new booking |
| GET | `/bookings` | List user's bookings |
| GET | `/studios` | List all studios (with filters) |
| GET | `/studios/:id` | Get studio details |
| GET | `/rooms/:id/availability` | Check room availability |
| GET | `/rooms/:id/busy-slots` | Get busy time slots |

---

## MWork Client Implementation

### 1. Create PhotoStudio Client Package

File: `internal/pkg/photostudio/booking_client.go`

```go
package photostudio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// BookingClient handles PhotoStudio booking API requests
type BookingClient struct {
	client  *Client // Reuse existing photostudio.Client
	baseURL string
	token   string
}

// NewBookingClient creates a booking client
func NewBookingClient(baseURL, token string, timeout time.Duration) *BookingClient {
	return &BookingClient{
		client:  NewClient(baseURL, token, timeout, "MWork/1.0.0"),
		baseURL: baseURL,
		token:   token,
	}
}

// CreateBookingRequest matches PhotoStudio's CreateBookingRequest
type CreateBookingRequest struct {
	RoomID    int64     `json:"room_id"`
	StudioID  int64     `json:"studio_id"`
	UserID    int64     `json:"user_id"`     // Will be auto-filled by middleware
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Notes     string    `json:"notes,omitempty"`
}

// BookingResponse matches PhotoStudio's BookingResponse
type BookingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Booking struct {
			ID         int64   `json:"id"`
			RoomID     int64   `json:"room_id"`
			RoomName   string  `json:"room_name,omitempty"`
			StudioID   int64   `json:"studio_id"`
			StudioName string  `json:"studio_name,omitempty"`
			StartTime  string  `json:"start_time"`
			EndTime    string  `json:"end_time"`
			Status     string  `json:"status"`
			TotalPrice float64 `json:"total_price"`
			Notes      string  `json:"notes,omitempty"`
			CreatedAt  string  `json:"created_at"`
		} `json:"booking"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// CreateBooking creates a booking with automatic user ID mapping
func (c *BookingClient) CreateBooking(
	ctx context.Context,
	mworkUserID uuid.UUID,
	req CreateBookingRequest,
) (*BookingResponse, error) {
	// Note: UserID in request will be ignored by PhotoStudio
	// The middleware extracts it from X-MWork-User-ID header
	
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/internal/mwork/bookings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Required headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("X-MWork-User-ID", mworkUserID.String())

	resp, err := c.client.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	var bookingResp BookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookingResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !bookingResp.Success {
		if bookingResp.Error != nil {
			return nil, fmt.Errorf("photostudio error: %s - %s", 
				bookingResp.Error.Code, bookingResp.Error.Message)
		}
		return nil, fmt.Errorf("booking failed with status %d", resp.StatusCode)
	}

	return &bookingResp, nil
}

// ListMyBookings retrieves user's bookings
func (c *BookingClient) ListMyBookings(
	ctx context.Context,
	mworkUserID uuid.UUID,
	limit, offset int,
) ([]interface{}, error) {
	url := fmt.Sprintf("%s/internal/mwork/bookings?limit=%d&offset=%d",
		c.baseURL, limit, offset)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("X-MWork-User-ID", mworkUserID.String())

	resp, err := c.client.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool          `json:"success"`
		Data    struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Items, nil
}

// GetRoomAvailability checks if a room is available
func (c *BookingClient) GetRoomAvailability(
	ctx context.Context,
	mworkUserID uuid.UUID,
	roomID int64,
	date string, // YYYY-MM-DD
) (interface{}, error) {
	url := fmt.Sprintf("%s/internal/mwork/rooms/%d/availability?date=%s",
		c.baseURL, roomID, date)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("X-MWork-User-ID", mworkUserID.String())

	resp, err := c.client.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ListStudios retrieves available studios
func (c *BookingClient) ListStudios(
	ctx context.Context,
	mworkUserID uuid.UUID,
	filters map[string]string,
) (interface{}, error) {
	url := c.baseURL + "/internal/mwork/studios"
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	q := httpReq.URL.Query()
	for k, v := range filters {
		q.Add(k, v)
	}
	httpReq.URL.RawQuery = q.Encode()

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("X-MWork-User-ID", mworkUserID.String())

	resp, err := c.client.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
```

---

### 2. MWork Handler Example

File: `internal/domain/photostudio/handler.go`

```go
package photostudio

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

type Handler struct {
	client *photostudio.BookingClient
}

func NewHandler(client *photostudio.BookingClient) *Handler {
	return &Handler{client: client}
}

// CreateBooking godoc
// @Summary Create a studio booking
// @Tags PhotoStudio
// @Security BearerAuth
// @Param request body CreateBookingDTO true "Booking details"
// @Success 201 {object} BookingResponseDTO
// @Router /api/v1/studio/bookings [post]
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// Extract MWork user ID from JWT context
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req CreateBookingDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.RoomID == 0 || req.StudioID == 0 {
		response.BadRequest(w, "room_id and studio_id are required")
		return
	}

	// Parse dates
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.BadRequest(w, "Invalid start_time format")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		response.BadRequest(w, "Invalid end_time format")
		return
	}

	// Create booking via PhotoStudio API
	photoReq := photostudio.CreateBookingRequest{
		RoomID:    req.RoomID,
		StudioID:  req.StudioID,
		StartTime: startTime,
		EndTime:   endTime,
		Notes:     req.Notes,
	}

	result, err := h.client.CreateBooking(r.Context(), userID, photoReq)
	if err != nil {
		response.InternalError(w, "Failed to create booking: "+err.Error())
		return
	}

	response.Created(w, result.Data.Booking)
}

// CreateBookingDTO is the request DTO for MWork
type CreateBookingDTO struct {
	RoomID    int64  `json:"room_id"`
	StudioID  int64  `json:"studio_id"`
	StartTime string `json:"start_time"` // ISO8601
	EndTime   string `json:"end_time"`   // ISO8601
	Notes     string `json:"notes,omitempty"`
}
```

---

### 3. Initialize in main.go

File: `cmd/api/main.go`

```go
// ... existing imports ...
import (
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
	photoStudioDomain "github.com/mwork/mwork-api/internal/domain/photostudio"
)

func main() {
	cfg := config.Load()
	
	// ... existing setup ...

	// PhotoStudio booking client
	photoStudioBookingClient := photostudio.NewBookingClient(
		cfg.PhotoStudioBaseURL,
		cfg.PhotoStudioToken,
		30*time.Second, // timeout
	)

	// PhotoStudio handler
	photoStudioHandler := photoStudioDomain.NewHandler(photoStudioBookingClient)

	// ... router setup ...

	r.Route("/api/v1/studio", func(r chi.Router) {
		r.Use(authMiddleware) // Requires JWT
		r.Post("/bookings", photoStudioHandler.CreateBooking)
		r.Get("/bookings", photoStudioHandler.ListMyBookings)
		r.Get("/studios", photoStudioHandler.ListStudios)
	})
}
```

---

## Configuration

### PhotoStudio (.env)

```env
# Port
PORT=8090

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/photostudio

# MWork Integration
MWORK_SYNC_ENABLED=true
MWORK_SYNC_TOKEN=your-super-secret-token-here
MWORK_SYNC_ALLOWED_IPS=127.0.0.1,localhost
```

### MWork (.env)

```env
# PhotoStudio Integration
PHOTO_STUDIO_BASE_URL=http://localhost:8090
PHOTO_STUDIO_TOKEN=your-super-secret-token-here
PHOTO_STUDIO_TIMEOUT_SECONDS=30
PHOTO_STUDIO_SYNC_ENABLED=true
```

---

## Example Request Flow

### 1. User logs in to MWork (gets JWT)
```http
POST http://localhost:8080/api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}

Response:
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "role": "model"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "..."
  }
}
```

### 2. User creates booking via MWork
```http
POST http://localhost:8080/api/v1/studio/bookings
Authorization: Bearer <MWork_JWT>
Content-Type: application/json

{
  "room_id": 1,
  "studio_id": 1,
  "start_time": "2026-02-15T10:00:00Z",
  "end_time": "2026-02-15T12:00:00Z",
  "notes": "Fashion photoshoot"
}
```

### 3. MWork → PhotoStudio (internal)
```http
POST http://localhost:8090/internal/mwork/bookings
Authorization: Bearer your-super-secret-token-here
X-MWork-User-ID: 550e8400-e29b-41d4-a716-446655440000
Content-Type: application/json

{
  "room_id": 1,
  "studio_id": 1,
  "start_time": "2026-02-15T10:00:00Z",
  "end_time": "2026-02-15T12:00:00Z",
  "notes": "Fashion photoshoot"
}
```

### 4. PhotoStudio Middleware Processing
```
1. Validate token: ✓
2. Extract X-MWork-User-ID: 550e8400-e29b-41d4-a716-446655440000
3. DB Query: SELECT * FROM users WHERE mwork_user_id = '550e8400-...'
4. Found: user { id: 42, email: "user@example.com", role: "client" }
5. Set context: c.Set("user_id", 42)
6. Forward to handler
```

### 5. BookingHandler receives context
```go
func (h *Handler) CreateBooking(c *gin.Context) {
    userID := c.GetInt64("user_id") // 42 ← PhotoStudio internal ID
    
    booking := &domain.Booking{
        UserID: userID, // Uses PhotoStudio ID
        RoomID: req.RoomID,
        // ...
    }
}
```

---

## Error Handling

### User Not Synced
```json
{
  "success": false,
  "error": {
    "code": "USER_NOT_SYNCED",
    "message": "User not found in PhotoStudio. Please contact support if this persists."
  }
}
```

**Solution**: Trigger manual sync or check if sync failed during registration.

### Invalid Token
```json
{
  "success": false,
  "error": {
    "code": "AUTH_INVALID",
    "message": "Invalid or missing internal token"
  }
}
```

**Solution**: Verify MWORK_SYNC_TOKEN matches in both .env files.

### Missing Header
```json
{
  "success": false,
  "error": {
    "code": "MWORK_USER_ID_MISSING",
    "message": "X-MWork-User-ID header is required"
  }
}
```

**Solution**: Ensure client sends X-MWork-User-ID header.

---

## Testing

### cURL Example
```bash
# Get PhotoStudio user ID
curl -X POST http://localhost:8090/internal/mwork/bookings \
  -H "Authorization: Bearer your-super-secret-token-here" \
  -H "X-MWork-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "studio_id": 1,
    "start_time": "2026-02-15T10:00:00Z",
    "end_time": "2026-02-15T12:00:00Z",
    "notes": "Test booking"
  }'
```

### Expected Response
```json
{
  "success": true,
  "data": {
    "booking": {
      "id": 123,
      "room_id": 1,
      "studio_id": 1,
      "status": "pending",
      "total_price": 5000.0,
      "start_time": "2026-02-15T10:00:00Z",
      "end_time": "2026-02-15T12:00:00Z"
    }
  }
}
```

---

## Summary

✅ **What's Implemented:**
1. MWorkUserAuth middleware - maps UUID → int64 ID
2. Internal booking routes - `/internal/mwork/bookings`
3. Token + header validation
4. Auto-population of user_id in Gin context

✅ **What MWork Needs:**
1. BookingClient wrapper (provided above)
2. Send `X-MWork-User-ID` header
3. Handle PhotoStudio responses

✅ **Zero Changes to:**
- BookingService logic
- Database schema
- Existing PhotoStudio endpoints

🎯 **Key Benefit:** MWork sends UUID, PhotoStudio automatically resolves to internal ID - transparent to business logic!
