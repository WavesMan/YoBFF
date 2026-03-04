export type HealthzResponse = {
  status?: string
}

export type CaptchaResponse = {
  captcha_id: string
  image_base64: string
  preset?: string
}

export type LogStats = {
  total?: number
  blocked?: number
  proxied?: number
  dropped?: number
}

export type LogLevelResponse = {
  level?: string
}

export type LogLevelUpdate = {
  level: string
}

export type DomainRule = {
  domain?: string
  upstream?: string
  forceHttps?: boolean
}

export type RoutingConfig = {
  defaultUpstream?: string
  domains?: DomainRule[]
}

export type SecurityConfig = {
  allowedCidrs?: string[]
  blockPageHtml?: string
  enableHsts?: boolean
}

export type Certificate = {
  domain?: string
}

export type CDNSyncConfig = {
  enabled?: boolean
  providers?: string[]
  schedule?: string
  cloudflare?: {
    ipv4_url?: string
    ipv6_url?: string
    endpoint?: string
    api_token?: string
    zone_id?: string
  }
}

export type CDNStatus = {
  provider?: string
  cidrs?: string[]
  last_sync?: string
  error?: string
}

export type Config = {
  dataPlane?: {
    httpListenAddr?: string
    httpsListenAddr?: string
    enableHttps?: boolean
  }
  controlPlane?: {
    adminListenAddr?: string
    auth?: {
      username?: string
      password?: string
      token?: string
    }
  }
  security?: SecurityConfig
  routing?: RoutingConfig
  certificates?: Certificate[]
  cdnSync?: CDNSyncConfig
}

export type CdnConfigResponse = {
  config?: CDNSyncConfig
  status?: Record<string, CDNStatus>
}

export type ApiError = {
  error_code?: string
  message?: string
  request_id?: string
  errors?: ValidationIssue[]
}

export type ValidationIssue = {
  path: string
  message: string
}

export type ConfigValidationResponse = {
  valid: boolean
  errors: ValidationIssue[]
}

export type ConfigVersion = {
  id: string
  created_at: string
  operator?: string
  source?: string
}

export type ConfigVersionResponse = {
  items: ConfigVersion[]
}

export type ConfigRollbackPayload = {
  version_id: string
}

export type ConfigRollbackResponse = {
  status?: string
  version_id?: string
}

export type LoginPayload = {
  username: string
  password: string
  captcha_id?: string
  captcha_code?: string
}

export type LoginResponse = {
  token?: string
}

export type LoginCaptchaRequirementResponse = {
  required: boolean
}
