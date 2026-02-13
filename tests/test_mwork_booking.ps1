# Test MWork Booking Integration (PowerShell)
# This script tests the X-MWork-User-ID header mapping

$PhotoStudioUrl = "http://localhost:8090"
$MWorkSyncToken = "your-super-secret-token-here"
$MWorkUserId = "550e8400-e29b-41d4-a716-446655440000"

Write-Host "🧪 Testing MWork → PhotoStudio Booking Integration" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

# Test 1: Create Booking
Write-Host "`nTest 1: Create Booking with X-MWork-User-ID" -ForegroundColor Yellow
Write-Host "--------------------------------------------" -ForegroundColor Yellow

$headers = @{
    "Authorization" = "Bearer $MWorkSyncToken"
    "X-MWork-User-ID" = $MWorkUserId
    "Content-Type" = "application/json"
}

$body = @{
    room_id = 1
    studio_id = 1
    start_time = "2026-02-15T10:00:00Z"
    end_time = "2026-02-15T12:00:00Z"
    notes = "Test booking via MWork"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/bookings" `
        -Method Post `
        -Headers $headers `
        -Body $body
    
    Write-Host "✅ SUCCESS" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5 | Write-Host
} catch {
    Write-Host "❌ FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 2: List My Bookings
Write-Host "`nTest 2: List User Bookings" -ForegroundColor Yellow
Write-Host "---------------------------" -ForegroundColor Yellow

$headers = @{
    "Authorization" = "Bearer $MWorkSyncToken"
    "X-MWork-User-ID" = $MWorkUserId
}

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/bookings?limit=10" `
        -Method Get `
        -Headers $headers
    
    Write-Host "✅ SUCCESS" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5 | Write-Host
} catch {
    Write-Host "❌ FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 3: Check Room Availability
Write-Host "`nTest 3: Check Room Availability" -ForegroundColor Yellow
Write-Host "--------------------------------" -ForegroundColor Yellow

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/rooms/1/availability?date=2026-02-15" `
        -Method Get `
        -Headers $headers
    
    Write-Host "✅ SUCCESS" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5 | Write-Host
} catch {
    Write-Host "❌ FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 4: List Studios
Write-Host "`nTest 4: List Studios" -ForegroundColor Yellow
Write-Host "--------------------" -ForegroundColor Yellow

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/studios?city=Almaty&limit=5" `
        -Method Get `
        -Headers $headers
    
    Write-Host "✅ SUCCESS" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5 | Write-Host
} catch {
    Write-Host "❌ FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 5: Error Case - Missing Header
Write-Host "`nTest 5: Error Case - Missing X-MWork-User-ID" -ForegroundColor Yellow
Write-Host "---------------------------------------------" -ForegroundColor Yellow

$headersNoUserId = @{
    "Authorization" = "Bearer $MWorkSyncToken"
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/bookings" `
        -Method Post `
        -Headers $headersNoUserId `
        -Body $body
    
    Write-Host "❌ Should have failed!" -ForegroundColor Red
} catch {
    Write-Host "✅ Expected error: $($_.Exception.Message)" -ForegroundColor Green
}

# Test 6: Error Case - Invalid Token
Write-Host "`nTest 6: Error Case - Invalid Token" -ForegroundColor Yellow
Write-Host "-----------------------------------" -ForegroundColor Yellow

$headersInvalidToken = @{
    "Authorization" = "Bearer invalid-token"
    "X-MWork-User-ID" = $MWorkUserId
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$PhotoStudioUrl/internal/mwork/bookings" `
        -Method Post `
        -Headers $headersInvalidToken `
        -Body $body
    
    Write-Host "❌ Should have failed!" -ForegroundColor Red
} catch {
    Write-Host "✅ Expected error: $($_.Exception.Message)" -ForegroundColor Green
}

Write-Host "`n✅ Tests completed!" -ForegroundColor Green
