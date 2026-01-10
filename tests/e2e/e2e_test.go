package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"photostudio/internal/database"
	"photostudio/internal/domain"
	"photostudio/internal/middleware"
	"photostudio/internal/modules/admin"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
	"photostudio/internal/modules/catalog"
	"photostudio/internal/modules/review"
	jwtsvc "photostudio/internal/pkg/jwt"
	"photostudio/internal/repository"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type E2ETestSuite struct {
	router      *gin.Engine
	db          *gorm.DB
	jwtService  *jwtsvc.Service
	testCleanup func()
}

type TestResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   *ErrorDetail           `json:"error,omitempty"`
	Message string                 `json:"message,omitempty"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func setupTestSuite(t *testing.T) *E2ETestSuite {
	// Use in-memory SQLite for testing
	testDB := ":memory:"

	db, err := database.Connect(testDB)
	require.NoError(t, err, "Failed to connect to test database")

	// Auto-migrate all models
	models := []interface{}{
		&domain.User{},
		&domain.StudioOwner{},
		&domain.Studio{},
		&domain.Room{},
		&domain.Equipment{},
		&domain.Booking{},
		&domain.Review{},
	}

	for _, model := range models {
		err := db.AutoMigrate(model)
		require.NoError(t, err, fmt.Sprintf("Failed to migrate %T", model))
	}

	// Setup repositories
	userRepo := repository.NewUserRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	equipmentRepo := repository.NewEquipmentRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	studioOwnerRepo := repository.NewStudioOwnerRepository(db)

	// Setup services
	jwtService := jwtsvc.New("test_secret_key_32_characters_min", 24*time.Hour)

	authService := auth.NewService(userRepo, studioOwnerRepo, jwtService)
	authHandler := auth.NewHandler(authService)

	catalogService := catalog.NewService(studioRepo, roomRepo, equipmentRepo)
	catalogHandler := catalog.NewHandler(catalogService, userRepo)

	bookingService := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingService)

	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)

	adminService := admin.NewService(userRepo, studioRepo, bookingRepo, reviewRepo)
	adminHandler := admin.NewHandler(adminService)

	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

	// Setup router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")

	// Public routes
	authHandler.RegisterPublicRoutes(v1)
	catalogHandler.RegisterRoutes(v1)
	reviewHandler.RegisterRoutes(v1, nil)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtService))
	{
		authHandler.RegisterProtectedRoutes(protected)
		bookingHandler.RegisterRoutes(protected)
		reviewHandler.RegisterRoutes(nil, protected)

		studios := protected.Group("/studios")
		{
			studios.GET("/my", middleware.RequireRole(string(domain.RoleStudioOwner)), catalogHandler.GetMyStudios)
			studios.POST("", middleware.RequireRole(string(domain.RoleStudioOwner)), catalogHandler.CreateStudio)
			studios.PUT("/:id", ownershipChecker.CheckStudioOwnership(), catalogHandler.UpdateStudio)
			studios.POST("/:id/rooms", ownershipChecker.CheckStudioOwnership(), catalogHandler.CreateRoom)
			studios.GET("/:id/bookings", middleware.RequireRole(string(domain.RoleStudioOwner)), ownershipChecker.CheckStudioOwnership(), bookingHandler.GetStudioBookings)
		}

		// Equipment route (only AddEquipment exists)
		rooms := protected.Group("/rooms")
		{
			rooms.POST("/:id/equipment", catalogHandler.AddEquipment)
		}

		adminGroup := protected.Group("/admin")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminHandler.RegisterRoutes(adminGroup)
		}

		bookings := protected.Group("/bookings")
		{
			bookings.PATCH("/:id/payment", middleware.RequireRole(string(domain.RoleStudioOwner)), bookingHandler.UpdatePaymentStatus)
		}
	}

	// Create admin user for testing
	adminUser := &domain.User{
		Email:         "admin@test.com",
		PasswordHash:  "$2a$10$dummy", // Will be properly hashed
		Role:          domain.RoleAdmin,
		Name:          "Admin User",
		Phone:         "+77001234560",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = db.Create(adminUser).Error
	require.NoError(t, err, "Failed to create admin user")

	return &E2ETestSuite{
		router:     r,
		db:         db,
		jwtService: jwtService,
		testCleanup: func() {
			// Cleanup is automatic with in-memory DB
		},
	}
}

func (s *E2ETestSuite) makeRequest(method, path string, body interface{}, token string) (*httptest.ResponseRecorder, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	return w, nil
}

func parseResponse(w *httptest.ResponseRecorder) (*TestResponse, error) {
	var resp TestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		// Print raw response for debugging
		log.Printf("Failed to parse response. Status: %d, Body: %s", w.Code, w.Body.String())
	}
	return &resp, err
}

func logErrorResponse(t *testing.T, resp *TestResponse, context string) {
	if resp.Error != nil {
		t.Logf("%s - Error: [%s] %s", context, resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			t.Logf("  Details: %+v", resp.Error.Details)
		}
	}
}

// Helper function to verify a studio owner (needed for studio creation)
func (s *E2ETestSuite) verifyStudioOwner(t *testing.T, email string) {
	// Find the user by email
	var user domain.User
	err := s.db.Where("email = ?", email).First(&user).Error
	require.NoError(t, err, "Failed to find user for verification")

	// Update user status to verified
	err = s.db.Model(&user).Update("studio_status", domain.StatusVerified).Error
	require.NoError(t, err, "Failed to verify studio owner")

	t.Logf("✅ Verified studio owner: %s", email)
}

// =============================================================================
// Test Flow 1: Client Registration and Authentication
// =============================================================================

func TestFlow1_ClientRegistrationAndAuth(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	t.Run("POST /auth/register/client", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "client@test.com",
			"password": "Password123!",
			"name":     "John Doe",
			"phone":    "+77001234567",
		}

		w, err := suite.makeRequest("POST", "/api/v1/auth/register/client", reqBody, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created")

		resp, err := parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Client registration failed")
		}
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data["token"])

		log.Printf("✅ POST /auth/register/client - SUCCESS")
	})

	t.Run("POST /auth/login", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "client@test.com",
			"password": "Password123!",
		}

		w, err := suite.makeRequest("POST", "/api/v1/auth/login", reqBody, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data["token"])

		log.Printf("✅ POST /auth/login - SUCCESS")
	})

	t.Run("GET /users/me", func(t *testing.T) {
		// First login to get token
		loginBody := map[string]interface{}{
			"email":    "client@test.com",
			"password": "Password123!",
		}

		loginResp, err := suite.makeRequest("POST", "/api/v1/auth/login", loginBody, "")
		require.NoError(t, err)

		loginData, err := parseResponse(loginResp)
		require.NoError(t, err)
		token := loginData.Data["token"].(string)

		// Now get user profile
		w, err := suite.makeRequest("GET", "/api/v1/users/me", nil, token)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Check if email is in top-level data or nested in user object
		if userMap, ok := resp.Data["user"].(map[string]interface{}); ok {
			assert.Equal(t, "client@test.com", userMap["email"])
		} else {
			assert.Equal(t, "client@test.com", resp.Data["email"])
		}

		log.Printf("✅ GET /users/me - SUCCESS")
	})
}

// =============================================================================
// Test Flow 2: Studio Search and Booking
// =============================================================================

func TestFlow2_SearchAndBooking(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	var clientToken, ownerToken string
	var studioID, roomID int64

	// Setup: Create client and studio owner
	t.Run("Setup: Create users", func(t *testing.T) {
		// Create client
		clientBody := map[string]interface{}{
			"email":    "client2@test.com",
			"name":     "Jane Smith",
			"password": "Password123!",
		}
		w, err := suite.makeRequest("POST", "/api/v1/auth/register/client", clientBody, "")
		require.NoError(t, err)
		resp, err := parseResponse(w)
		require.NoError(t, err)
		clientToken = resp.Data["token"].(string)

		// Create studio owner
		ownerBody := map[string]interface{}{
			"email":        "owner@test.com",
			"name":         "Studio Owner",
			"password":     "Password123!@",
			"company_name": "Best Studios",
			"bin":          "123456789012",
			"phone":        "+77001234567",
		}
		w, err = suite.makeRequest("POST", "/api/v1/auth/register/studio", ownerBody, "")
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		ownerToken = resp.Data["token"].(string)

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, ownerBody["email"].(string))

		// Create studio
		studioBody := map[string]interface{}{
			"name":        "Test Studio",
			"description": "A great photo studio",
			"address":     "123 Main St",
			"district":    "Almalinsky",
			"city":        "Almaty",
			"phone":       "+7 727 123 4567",
		}
		w, err = suite.makeRequest("POST", "/api/v1/studios", studioBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Studio creation failed")
			t.FailNow()
		}
		// Extract studio ID from nested structure
		if studioData, ok := resp.Data["studio"].(map[string]interface{}); ok {
			if idVal, ok := studioData["id"]; ok && idVal != nil {
				studioID = int64(idVal.(float64))
			} else {
				t.Fatal("Studio data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			studioID = int64(idVal.(float64))
		} else {
			t.Fatal("Studio creation succeeded but no ID returned")
		}

		// Create room
		roomBody := map[string]interface{}{
			"name":               "Main Room",
			"description":        "Spacious photo room",
			"capacity":           10,
			"area_sqm":           50,
			"room_type":          "Portrait",
			"price_per_hour_min": 5000.0,
		}
		w, err = suite.makeRequest("POST", fmt.Sprintf("/api/v1/studios/%d/rooms", studioID), roomBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Room creation failed")
			t.FailNow()
		}
		// Extract room ID from nested structure
		if roomData, ok := resp.Data["room"].(map[string]interface{}); ok {
			if idVal, ok := roomData["id"]; ok && idVal != nil {
				roomID = int64(idVal.(float64))
			} else {
				t.Fatal("Room data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			roomID = int64(idVal.(float64))
		} else {
			t.Fatal("Room creation succeeded but no ID returned")
		}
	})

	t.Run("GET /studios", func(t *testing.T) {
		w, err := suite.makeRequest("GET", "/api/v1/studios", nil, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /studios - SUCCESS")
	})

	t.Run("GET /studios/:id", func(t *testing.T) {
		w, err := suite.makeRequest("GET", fmt.Sprintf("/api/v1/studios/%d", studioID), nil, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "Test Studio", resp.Data["name"])

		log.Printf("✅ GET /studios/:id - SUCCESS")
	})

	t.Run("GET /rooms/:id/availability", func(t *testing.T) {
		startDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		endDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")

		w, err := suite.makeRequest("GET", fmt.Sprintf("/api/v1/rooms/%d/availability?start_date=%s&end_date=%s", roomID, startDate, endDate), nil, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /rooms/:id/availability - SUCCESS")
	})

	var bookingID int64
	t.Run("POST /bookings", func(t *testing.T) {
		startTime := time.Now().Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)

		bookingBody := map[string]interface{}{
			"room_id":    roomID,
			"start_time": startTime.Format(time.RFC3339),
			"end_time":   endTime.Format(time.RFC3339),
		}

		w, err := suite.makeRequest("POST", "/api/v1/bookings", bookingBody, clientToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Safely extract booking ID
		if idVal, ok := resp.Data["id"]; ok {
			bookingID = int64(idVal.(float64))
		}

		log.Printf("✅ POST /bookings - SUCCESS")
	})

	t.Run("GET /users/me/bookings", func(t *testing.T) {
		w, err := suite.makeRequest("GET", "/api/v1/users/me/bookings", nil, clientToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /users/me/bookings - SUCCESS (booking_id: %d)", bookingID)
	})
}

// =============================================================================
// Test Flow 3: Studio Owner Operations
// =============================================================================

func TestFlow3_StudioOwnerOperations(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	var ownerToken, clientToken string
	var studioID, roomID, bookingID int64

	t.Run("POST /auth/register/studio", func(t *testing.T) {
		ownerBody := map[string]interface{}{
			"email":        "newowner@test.com",
			"name":         "New Owner",
			"password":     "Password123!@",
			"company_name": "New Studios Inc",
			"bin":          "123456789012",
			"phone":        "+77001234568",
		}

		w, err := suite.makeRequest("POST", "/api/v1/auth/register/studio", ownerBody, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		ownerToken = resp.Data["token"].(string)

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, "newowner@test.com")

		log.Printf("✅ POST /auth/register/studio - SUCCESS")
	})

	// Create studio and room for booking test
	t.Run("Setup: Create studio and room", func(t *testing.T) {
		studioBody := map[string]interface{}{
			"name":        "Owner Studio",
			"description": "Test studio for owner",
			"address":     "456 Test Ave",
			"district":    "Medeusky",
			"city":        "Almaty",
			"phone":       "+7 727 456 7890",
		}
		w, err := suite.makeRequest("POST", "/api/v1/studios", studioBody, ownerToken)
		require.NoError(t, err)
		resp, err := parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Studio creation failed")
			t.FailNow()
		}
		// Extract studio ID from nested structure
		if studioData, ok := resp.Data["studio"].(map[string]interface{}); ok {
			if idVal, ok := studioData["id"]; ok && idVal != nil {
				studioID = int64(idVal.(float64))
			} else {
				t.Fatal("Studio data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			studioID = int64(idVal.(float64))
		} else {
			t.Fatal("Studio creation succeeded but no ID returned")
		}

		roomBody := map[string]interface{}{
			"name":               "Test Room",
			"description":        "Room for testing",
			"capacity":           5,
			"area_sqm":           30,
			"room_type":          "Creative",
			"price_per_hour_min": 3000.0,
		}
		w, err = suite.makeRequest("POST", fmt.Sprintf("/api/v1/studios/%d/rooms", studioID), roomBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Room creation failed")
			t.FailNow()
		}
		// Extract room ID from nested structure
		if roomData, ok := resp.Data["room"].(map[string]interface{}); ok {
			if idVal, ok := roomData["id"]; ok && idVal != nil {
				roomID = int64(idVal.(float64))
			} else {
				t.Fatal("Room data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			roomID = int64(idVal.(float64))
		} else {
			t.Fatal("Room creation succeeded but no ID returned")
		}

		// Create client and booking
		clientBody := map[string]interface{}{
			"email":    "client3@test.com",
			"name":     "Test Client",
			"password": "Password123!",
		}
		w, err = suite.makeRequest("POST", "/api/v1/auth/register/client", clientBody, "")
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		clientToken = resp.Data["token"].(string)

		bookingBody := map[string]interface{}{
			"room_id":    roomID,
			"start_time": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"end_time":   time.Now().Add(26 * time.Hour).Format(time.RFC3339),
		}
		w, err = suite.makeRequest("POST", "/api/v1/bookings", bookingBody, clientToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Booking creation failed")
			t.FailNow()
		}
		if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			bookingID = int64(idVal.(float64))
		} else {
			t.Fatal("Booking creation succeeded but no ID returned")
		}
	})

	t.Run("GET /studios/my", func(t *testing.T) {
		w, err := suite.makeRequest("GET", "/api/v1/studios/my", nil, ownerToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /studios/my - SUCCESS")
	})

	t.Run("PATCH /bookings/:id/status", func(t *testing.T) {
		statusBody := map[string]interface{}{
			"status": "confirmed",
		}

		w, err := suite.makeRequest("PATCH", fmt.Sprintf("/api/v1/bookings/%d/status", bookingID), statusBody, ownerToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ PATCH /bookings/:id/status - SUCCESS")
	})
}

// =============================================================================
// Test Flow 4: Admin Operations
// =============================================================================

func TestFlow4_AdminOperations(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	var adminToken, ownerToken string
	var studioID int64

	// Get admin token
	t.Run("Setup: Get admin token", func(t *testing.T) {
		// Manually create admin user and generate token
		adminUser := &domain.User{}
		err := suite.db.Where("email = ?", "admin@test.com").First(adminUser).Error
		require.NoError(t, err)

		adminToken, err = suite.jwtService.GenerateToken(adminUser.ID, string(adminUser.Role))
		require.NoError(t, err)
	})

	// Create studio owner and pending studio
	t.Run("Setup: Create pending studio", func(t *testing.T) {
		ownerBody := map[string]interface{}{
			"email":        "pendingowner@test.com",
			"name":         "Pending Owner",
			"password":     "Password123!@",
			"company_name": "Pending Studios",
			"bin":          "123456789012",
			"phone":        "+77001234569",
		}
		w, err := suite.makeRequest("POST", "/api/v1/auth/register/studio", ownerBody, "")
		require.NoError(t, err)
		resp, err := parseResponse(w)
		require.NoError(t, err)
		ownerToken = resp.Data["token"].(string)

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, "pendingowner@test.com")

		studioBody := map[string]interface{}{
			"name":        "Pending Studio",
			"description": "Awaiting verification",
			"address":     "789 Pending St",
			"district":    "Auezovsky",
			"city":        "Almaty",
			"phone":       "+7 727 789 0123",
		}
		w, err = suite.makeRequest("POST", "/api/v1/studios", studioBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Studio creation failed")
			t.FailNow()
		}
		// Extract studio ID from nested structure
		if studioData, ok := resp.Data["studio"].(map[string]interface{}); ok {
			if idVal, ok := studioData["id"]; ok && idVal != nil {
				studioID = int64(idVal.(float64))
			} else {
				t.Fatal("Studio data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			studioID = int64(idVal.(float64))
		} else {
			t.Fatal("Studio creation succeeded but no ID returned")
		}
	})

	t.Run("GET /admin/studios/pending", func(t *testing.T) {
		w, err := suite.makeRequest("GET", "/api/v1/admin/studios/pending", nil, adminToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /admin/studios/pending - SUCCESS")
	})

	t.Run("POST /admin/studios/:id/verify", func(t *testing.T) {
		verifyBody := map[string]interface{}{
			"verified": true,
		}

		w, err := suite.makeRequest("POST", fmt.Sprintf("/api/v1/admin/studios/%d/verify", studioID), verifyBody, adminToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ POST /admin/studios/:id/verify - SUCCESS")
	})

	t.Run("GET /admin/statistics", func(t *testing.T) {
		w, err := suite.makeRequest("GET", "/api/v1/admin/statistics", nil, adminToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /admin/statistics - SUCCESS")
	})
}

// =============================================================================
// Test Flow 5: Equipment Management
// =============================================================================

func TestFlow5_EquipmentManagement(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	var ownerToken string
	var studioID, roomID, equipmentID int64

	// Setup: Create owner, studio, and room
	t.Run("Setup: Create owner and studio", func(t *testing.T) {
		ownerBody := map[string]interface{}{
			"email":        "equipowner@test.com",
			"name":         "Equipment Owner",
			"password":     "Password123!@",
			"company_name": "Equipment Studios",
			"bin":          "123456789012",
			"phone":        "+77001234570",
		}
		w, err := suite.makeRequest("POST", "/api/v1/auth/register/studio", ownerBody, "")
		require.NoError(t, err)
		resp, err := parseResponse(w)
		require.NoError(t, err)
		ownerToken = resp.Data["token"].(string)

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, "equipowner@test.com")

		studioBody := map[string]interface{}{
			"name":        "Equipment Test Studio",
			"description": "Studio for equipment testing",
			"address":     "100 Equipment St",
			"district":    "Almalinsky",
			"city":        "Almaty",
			"phone":       "+7 727 100 0000",
		}
		w, err = suite.makeRequest("POST", "/api/v1/studios", studioBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Studio creation failed")
			t.FailNow()
		}
		// Extract studio ID from nested structure
		if studioData, ok := resp.Data["studio"].(map[string]interface{}); ok {
			if idVal, ok := studioData["id"]; ok && idVal != nil {
				studioID = int64(idVal.(float64))
			} else {
				t.Fatal("Studio data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			studioID = int64(idVal.(float64))
		} else {
			t.Fatal("Studio creation succeeded but no ID returned")
		}

		roomBody := map[string]interface{}{
			"name":               "Equipment Room",
			"description":        "Room with equipment",
			"capacity":           8,
			"area_sqm":           40,
			"room_type":          "Fashion",
			"price_per_hour_min": 4000.0,
		}
		w, err = suite.makeRequest("POST", fmt.Sprintf("/api/v1/studios/%d/rooms", studioID), roomBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Room creation failed")
			t.FailNow()
		}
		// Extract room ID from nested structure
		if roomData, ok := resp.Data["room"].(map[string]interface{}); ok {
			if idVal, ok := roomData["id"]; ok && idVal != nil {
				roomID = int64(idVal.(float64))
			} else {
				t.Fatal("Room data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			roomID = int64(idVal.(float64))
		} else {
			t.Fatal("Room creation succeeded but no ID returned")
		}
	})

	t.Run("POST /rooms/:id/equipment", func(t *testing.T) {
		equipmentBody := map[string]interface{}{
			"name":         "Canon EOS R5",
			"category":     "Camera",
			"brand":        "Canon",
			"model":        "EOS R5",
			"quantity":     2,
			"rental_price": 5000.0,
		}

		w, err := suite.makeRequest("POST", fmt.Sprintf("/api/v1/rooms/%d/equipment", roomID), equipmentBody, ownerToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		if idVal, ok := resp.Data["id"]; ok {
			equipmentID = int64(idVal.(float64))
		}

		log.Printf("✅ POST /rooms/:id/equipment - SUCCESS (equipment_id: %d)", equipmentID)
	})

	t.Run("GET /rooms/:id/equipment", func(t *testing.T) {
		t.Skip("GET equipment endpoint not implemented yet")
		w, err := suite.makeRequest("GET", fmt.Sprintf("/api/v1/rooms/%d/equipment", roomID), nil, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /rooms/:id/equipment - SUCCESS")
	})

	t.Run("PUT /equipment/:id", func(t *testing.T) {
		t.Skip("PUT equipment endpoint not implemented yet")
		updateBody := map[string]interface{}{
			"quantity":     3,
			"rental_price": 5500.0,
		}

		w, err := suite.makeRequest("PUT", fmt.Sprintf("/api/v1/equipment/%d", equipmentID), updateBody, ownerToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ PUT /equipment/:id - SUCCESS")
	})

	t.Run("DELETE /equipment/:id", func(t *testing.T) {
		t.Skip("DELETE equipment endpoint not implemented yet")
		w, err := suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/equipment/%d", equipmentID), nil, ownerToken)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ DELETE /equipment/:id - SUCCESS")
	})
}

// =============================================================================
// Test Flow 6: Review System
// =============================================================================

func TestFlow6_ReviewSystem(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.testCleanup()

	var clientToken, ownerToken string
	var studioID, roomID, bookingID, reviewID int64

	// Setup: Create complete booking scenario
	t.Run("Setup: Create booking scenario", func(t *testing.T) {
		// Create client
		clientBody := map[string]interface{}{
			"email":    "reviewclient@test.com",
			"name":     "Review Client",
			"password": "Password123!",
		}
		w, err := suite.makeRequest("POST", "/api/v1/auth/register/client", clientBody, "")
		require.NoError(t, err)
		resp, err := parseResponse(w)
		require.NoError(t, err)
		clientToken = resp.Data["token"].(string)

		// Create owner
		ownerBody := map[string]interface{}{
			"email":        "reviewowner@test.com",
			"name":         "Review Owner",
			"password":     "Password123!@",
			"company_name": "Review Studios",
			"bin":          "123456789012",
			"phone":        "+77001234571",
		}
		w, err = suite.makeRequest("POST", "/api/v1/auth/register/studio", ownerBody, "")
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		ownerToken = resp.Data["token"].(string)

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, "reviewowner@test.com")

		// Verify the studio owner so they can create studios
		suite.verifyStudioOwner(t, ownerBody["email"].(string))

		// Create studio
		studioBody := map[string]interface{}{
			"name":        "Review Test Studio",
			"description": "Studio for review testing",
			"address":     "200 Review St",
			"district":    "Bostandyksky",
			"city":        "Almaty",
			"phone":       "+7 727 200 0000",
		}
		w, err = suite.makeRequest("POST", "/api/v1/studios", studioBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Studio creation failed")
			t.FailNow()
		}
		// Extract studio ID from nested structure
		if studioData, ok := resp.Data["studio"].(map[string]interface{}); ok {
			if idVal, ok := studioData["id"]; ok && idVal != nil {
				studioID = int64(idVal.(float64))
			} else {
				t.Fatal("Studio data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			studioID = int64(idVal.(float64))
		} else {
			t.Fatal("Studio creation succeeded but no ID returned")
		}

		// Create room
		roomBody := map[string]interface{}{
			"name":               "Review Room",
			"description":        "Room for review",
			"capacity":           6,
			"area_sqm":           35,
			"room_type":          "Commercial",
			"price_per_hour_min": 3500.0,
		}
		w, err = suite.makeRequest("POST", fmt.Sprintf("/api/v1/studios/%d/rooms", studioID), roomBody, ownerToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Room creation failed")
			t.FailNow()
		}
		// Extract room ID from nested structure
		if roomData, ok := resp.Data["room"].(map[string]interface{}); ok {
			if idVal, ok := roomData["id"]; ok && idVal != nil {
				roomID = int64(idVal.(float64))
			} else {
				t.Fatal("Room data exists but no ID field")
			}
		} else if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			roomID = int64(idVal.(float64))
		} else {
			t.Fatal("Room creation succeeded but no ID returned")
		}

		// Create completed booking
		bookingBody := map[string]interface{}{
			"room_id":    roomID,
			"start_time": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			"end_time":   time.Now().Add(-46 * time.Hour).Format(time.RFC3339),
		}
		w, err = suite.makeRequest("POST", "/api/v1/bookings", bookingBody, clientToken)
		require.NoError(t, err)
		resp, err = parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Booking creation failed")
			t.FailNow()
		}
		if idVal, ok := resp.Data["id"]; ok && idVal != nil {
			bookingID = int64(idVal.(float64))
		} else {
			t.Fatal("Booking creation succeeded but no ID returned")
		}

		// Mark booking as completed
		statusBody := map[string]interface{}{
			"status": "completed",
		}
		w, err = suite.makeRequest("PATCH", fmt.Sprintf("/api/v1/bookings/%d/status", bookingID), statusBody, ownerToken)
		require.NoError(t, err)
	})

	t.Run("POST /reviews", func(t *testing.T) {
		reviewBody := map[string]interface{}{
			"studio_id":  studioID,
			"booking_id": bookingID,
			"rating":     5,
			"comment":    "Excellent studio! Professional equipment and great service.",
		}

		w, err := suite.makeRequest("POST", "/api/v1/reviews", reviewBody, clientToken)
		require.NoError(t, err)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Review creation failed")
		}

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.True(t, resp.Success)

		if idVal, ok := resp.Data["id"]; ok {
			reviewID = int64(idVal.(float64))
		}

		log.Printf("✅ POST /reviews - SUCCESS (review_id: %d)", reviewID)
	})

	t.Run("GET /studios/:id/reviews", func(t *testing.T) {
		w, err := suite.makeRequest("GET", fmt.Sprintf("/api/v1/studios/%d/reviews", studioID), nil, "")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		log.Printf("✅ GET /studios/:id/reviews - SUCCESS")
	})

	t.Run("POST /reviews/:id/response", func(t *testing.T) {
		responseBody := map[string]interface{}{
			"response": "Thank you for your feedback! We're glad you enjoyed your experience.",
		}

		w, err := suite.makeRequest("POST", fmt.Sprintf("/api/v1/reviews/%d/response", reviewID), responseBody, ownerToken)
		require.NoError(t, err)

		resp, err := parseResponse(w)
		require.NoError(t, err)
		if !resp.Success {
			logErrorResponse(t, resp, "Owner response failed")
		}

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, resp.Success)

		log.Printf("✅ POST /reviews/:id/response - SUCCESS")
	})
}

// =============================================================================
// Main Test Runner
// =============================================================================

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
