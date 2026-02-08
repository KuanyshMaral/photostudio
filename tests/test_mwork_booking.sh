#!/usr/bin/env bash

# Test MWork Booking Integration
# This script tests the X-MWork-User-ID header mapping

set -e

PHOTOSTUDIO_URL="http://localhost:8090"
MWORK_SYNC_TOKEN="your-super-secret-token-here"
MWORK_USER_ID="550e8400-e29b-41d4-a716-446655440000"

echo "🧪 Testing MWork → PhotoStudio Booking Integration"
echo "=================================================="

# Test 1: Create Booking
echo ""
echo "Test 1: Create Booking with X-MWork-User-ID"
echo "--------------------------------------------"

curl -X POST "${PHOTOSTUDIO_URL}/internal/mwork/bookings" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -H "X-MWork-User-ID: ${MWORK_USER_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "studio_id": 1,
    "start_time": "2026-02-15T10:00:00Z",
    "end_time": "2026-02-15T12:00:00Z",
    "notes": "Test booking via MWork"
  }' \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

# Test 2: List My Bookings
echo ""
echo "Test 2: List User Bookings"
echo "---------------------------"

curl -X GET "${PHOTOSTUDIO_URL}/internal/mwork/bookings?limit=10" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -H "X-MWork-User-ID: ${MWORK_USER_ID}" \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

# Test 3: Check Room Availability
echo ""
echo "Test 3: Check Room Availability"
echo "--------------------------------"

curl -X GET "${PHOTOSTUDIO_URL}/internal/mwork/rooms/1/availability?date=2026-02-15" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -H "X-MWork-User-ID: ${MWORK_USER_ID}" \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

# Test 4: List Studios
echo ""
echo "Test 4: List Studios"
echo "--------------------"

curl -X GET "${PHOTOSTUDIO_URL}/internal/mwork/studios?city=Almaty&limit=5" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -H "X-MWork-User-ID: ${MWORK_USER_ID}" \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

# Test 5: Error Case - Missing Header
echo ""
echo "Test 5: Error Case - Missing X-MWork-User-ID"
echo "---------------------------------------------"

curl -X POST "${PHOTOSTUDIO_URL}/internal/mwork/bookings" \
  -H "Authorization: Bearer ${MWORK_SYNC_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "studio_id": 1,
    "start_time": "2026-02-15T14:00:00Z",
    "end_time": "2026-02-15T16:00:00Z"
  }' \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

# Test 6: Error Case - Invalid Token
echo ""
echo "Test 6: Error Case - Invalid Token"
echo "-----------------------------------"

curl -X POST "${PHOTOSTUDIO_URL}/internal/mwork/bookings" \
  -H "Authorization: Bearer invalid-token" \
  -H "X-MWork-User-ID: ${MWORK_USER_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "room_id": 1,
    "studio_id": 1,
    "start_time": "2026-02-15T14:00:00Z",
    "end_time": "2026-02-15T16:00:00Z"
  }' \
  -w "\n\nHTTP Status: %{http_code}\n" \
  | jq '.'

echo ""
echo "✅ Tests completed!"
