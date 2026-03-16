Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Invoke-GoQualityGate runs backend quality checks.
function Invoke-GoQualityGate {
    go vet ./...
    go test ./... -coverprofile=coverage.out
    if (-not (Test-Path "coverage.out")) {
        throw "coverage.out was not generated, coverage check failed."
    }
    $coverLine = go tool cover -func coverage.out | Select-Object -Last 1
    if ([string]::IsNullOrWhiteSpace($coverLine)) {
        throw "failed to read coverage summary."
    }
    $rateText = (($coverLine -split "\s+")[-1]).TrimEnd("%")
    $rate = [double]$rateText
    $coverageThreshold = 75
    if ($rate -lt $coverageThreshold) {
        throw "Go coverage is below threshold: $rate%, expected at least $coverageThreshold%."
    }
    go test ./contracts/openapi -run TestAdminOpenAPIContractConsistency -v
    go test ./internal/admin -run TestWeaverDraftRoutes -v
}

# Invoke-WebQualityGate runs frontend quality checks.
function Invoke-WebQualityGate {
    Push-Location "web/source"
    try {
        pnpm install --frozen-lockfile
        pnpm lint
        pnpm test -- --run
        pnpm build
    } finally {
        Pop-Location
    }
}

# Invoke-SonarQualityGate runs SonarQube checks when env vars are set.
function Invoke-SonarQualityGate {
    if ([string]::IsNullOrWhiteSpace($env:SONAR_HOST_URL) -or
        [string]::IsNullOrWhiteSpace($env:SONAR_TOKEN) -or
        [string]::IsNullOrWhiteSpace($env:SONAR_PROJECT_KEY)) {
        Write-Host "SONAR_* environment variables are missing, skipping SonarQube checks."
        return
    }
    sonar-scanner `
        "-Dsonar.host.url=$($env:SONAR_HOST_URL)" `
        "-Dsonar.token=$($env:SONAR_TOKEN)" `
        "-Dsonar.projectKey=$($env:SONAR_PROJECT_KEY)" `
        "-Dsonar.qualitygate.wait=true"
}

Invoke-GoQualityGate
Invoke-WebQualityGate
Invoke-SonarQualityGate
Write-Host "Quality gate passed."
