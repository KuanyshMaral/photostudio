# PhotoStudio Direct Booking Test
# Tests PhotoStudio booking API directly with X-MWork-User-ID header

$ErrorActionPreference = "Stop"

$PhotoStudioURL = "http://localhost:8090"
$MWorkSyncToken = "your-super-secret-token-here"
$TestUserUUID = "550e8400-e29b-41d4-a716-446655440000"

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  PhotoStudio Direct Booking Test" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

function Write-Success {
    Write-Host "OK: $args" -ForegroundColor Green
}

function Write-Failure {
    Write-Host "FAIL: $args" -ForegroundColor Red
}

function Write-Info {
    Write-Host "INFO: $args" -ForegroundColor Cyan
}

try {
    # Check PhotoStudio
    Write-Info "Checking PhotoStudio..."
    
    try {
        $health = Invoke-RestMethod -Uri "$PhotoStudioURL/healthz" -Method Get -TimeoutSec 3
        Write-Success "PhotoStudio is running"
    } catch {
        Write-Failure "PhotoStudio is not running on $PhotoStudioURL"
        Write-Host "`nStart PhotoStudio with:" -ForegroundColor Yellow
        Write-Host "  cd photostudio-main" -ForegroundColor Gray
        Write-Host "  go run cmd/api/main.go" -ForegroundColor Gray
        exit 1
    }

    # Get studios
    Write-Host "`n----" -ForegroundColor Cyan
    Write-Info "Getting studios list..."
    
    try {
        $studiosResponse = Invoke-RestMethod -Uri "$PhotoStudioURL/api/v1/studios?limit=10" -Method Get -TimeoutSec 10
        Write-Success "Studios retrieved"
    } catch {
        Write-Host "Studios endpoint not ready (using default IDs)" -ForegroundColor Yellow
    }

    # Create booking
    Write-Host "`n----" -ForegroundColor Cyan
    Write-Info "Creating booking via internal API..."
    
    $tomorrow = (Get-Date).AddDays(1)
    $startTime = Get-Date -Year $tomorrow.Year -Month $tomorrow.Month -Day $tomorrow.Day -Hour 14 -Minute 0 -Second 0
    $endTime = $startTime.AddHours(2)
    
    $startTimeStr = $startTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $endTimeStr = $endTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

    $headers = @{
        "Authorization" = "Bearer $MWorkSyncToken"
        "X-MWork-User-ID" = $TestUserUUID
        "Content-Type" = "application/json"
    }

    $bookingBody = @{
        studio_id = 1
        room_id = 1
        start_time = $startTimeStr
        end_time = $endTimeStr
        notes = "Direct API test"
    } | ConvertTo-Json

    Write-Host "Booking time: $startTimeStr - $endTimeStr" -ForegroundColor Gray
    Write-Host "User UUID: $TestUserUUID" -ForegroundColor Gray

    try {
        $bookingResponse = Invoke-RestMethod `
            -Uri "$PhotoStudioURL/internal/mwork/bookings" `
            -Method Post `
            -Headers $headers `
            -Body $bookingBody `
            -TimeoutSec 10

        Write-Success "Booking created"
        Write-Host "Booking ID: $($bookingResponse.data.booking.id)" -ForegroundColor Gray
        
    } catch {
        Write-Failure "Booking creation failed"
        
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
            Write-Host "Status: $statusCode" -ForegroundColor Yellow
            
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $responseBody = $reader.ReadToEnd()
                $reader.Close()
                
                Write-Host "Response: $responseBody" -ForegroundColor Yellow
            } catch {
                # silent
            }
        }
        
        throw
    }

    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "  SUCCESS - Test Passed!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "`nIntegration verified:" -ForegroundColor Green
    Write-Host "  - PhotoStudio running" -ForegroundColor Green
    Write-Host "  - X-MWork-User-ID header works" -ForegroundColor Green
    Write-Host "  - Booking created successfully" -ForegroundColor Green
    Write-Host "`n" -ForegroundColor Green

} catch {
    Write-Host "`n========================================" -ForegroundColor Red
    Write-Host "  TEST FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "`nError: $($_.Exception.Message)`n" -ForegroundColor Red
    exit 1
}
