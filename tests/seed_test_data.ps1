# Seed PhotoStudio with test data

$PhotoStudioURL = "http://localhost:8090"
$token = "your-super-secret-token-here"

Write-Host "Creating test studio..." -ForegroundColor Cyan

$studioBody = @{
    name = "Test Studio"
    city = "Almaty"
    address = "123 Test St"
    description = "Test studio for integration testing"
    phone = "+7-700-123-4567"
    email = "test@studio.com"
    price_per_hour = 50000
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod `
        -Uri "$PhotoStudioURL/internal/admin/studios" `
        -Method Post `
        -Headers @{
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
        } `
        -Body $studioBody `
        -TimeoutSec 10
    
    $studioId = $response.data.studio.id
    Write-Host "Studio created: ID=$studioId" -ForegroundColor Green
    
    # Create room
    Write-Host "`nCreating test room..." -ForegroundColor Cyan
    
    $roomBody = @{
        studio_id = $studioId
        name = "Main Hall"
        capacity = 100
        price_per_hour = 50000
        description = "Main photography hall"
    } | ConvertTo-Json
    
    $roomResponse = Invoke-RestMethod `
        -Uri "$PhotoStudioURL/internal/admin/rooms" `
        -Method Post `
        -Headers @{
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
        } `
        -Body $roomBody `
        -TimeoutSec 10
    
    Write-Host "Room created: ID=$($roomResponse.data.room.id)" -ForegroundColor Green
    Write-Host "`nTest data seeded successfully!" -ForegroundColor Green
    Write-Host "Studio ID: $studioId" -ForegroundColor Gray
    
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
