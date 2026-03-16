Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Invoke-GoQualityGate 执行后端质量门禁，包括静态检查、测试与覆盖率阈值校验。
function Invoke-GoQualityGate {
    go vet ./...
    go test ./... -coverprofile=coverage.out
    if (-not (Test-Path "coverage.out")) {
        throw "未生成 coverage.out，覆盖率校验失败。"
    }
    $coverLine = go tool cover -func=coverage.out | Select-Object -Last 1
    if ([string]::IsNullOrWhiteSpace($coverLine)) {
        throw "无法读取覆盖率统计。"
    }
    $rateText = (($coverLine -split "\s+")[-1]).TrimEnd("%")
    $rate = [double]$rateText
    if ($rate -lt 80) {
        throw "Go 覆盖率不达标：$rate%，要求至少 80%。"
    }
    go test ./contracts/openapi -run TestAdminOpenAPIContractConsistency -v
    go test ./internal/admin -run TestWeaverDraftRoutes -v
}

# Invoke-WebQualityGate 执行前端质量门禁，包括依赖锁定安装、Lint、测试与构建。
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

# Invoke-SonarQualityGate 在环境变量齐全时执行 SonarQube 扫描与质量门禁等待。
function Invoke-SonarQualityGate {
    if ([string]::IsNullOrWhiteSpace($env:SONAR_HOST_URL) -or
        [string]::IsNullOrWhiteSpace($env:SONAR_TOKEN) -or
        [string]::IsNullOrWhiteSpace($env:SONAR_PROJECT_KEY)) {
        Write-Host "未配置 SONAR_* 环境变量，跳过 SonarQube 校验。"
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
Write-Host "质量门禁通过。"
