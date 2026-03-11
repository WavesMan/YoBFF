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

export type LBNode = {
  id?: string
  upstream?: string
  weight?: number
  enabled?: boolean
}

export type LBPool = {
  id?: string
  name?: string
  strategy?: string
  nodes?: LBNode[]
}

export type LBRouteRule = {
  domain?: string
  poolId?: string
  fallbackPoolId?: string
  forceHttps?: boolean
}

export type LoadBalancerConfig = {
  defaultPoolId?: string
  pools?: LBPool[]
  routes?: LBRouteRule[]
}

export type CDNProviderSetting = {
  apiKey?: string
  secretKey?: string
  option?: string
}

export type SecurityConfig = {
  allowedCidrs?: string[]
  trustedProxyCidrs?: string[]
  allowedCdnProviders?: string[]
  cdnProviderSettings?: Record<string, CDNProviderSetting>
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

export type RateLimitConfig = {
  enabled?: boolean
  requestsPerSecond?: number
  burst?: number
}

export type SSLCertificate = {
  id: string
  name: string
  domains: string[]
  notAfter: string
  issuer: string
  createdAt: string
}

export type Config = {
  dataPlane?: {
    httpListenAddr?: string
    httpsListenAddr?: string
    enableHttps?: boolean
    certId?: string
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
  loadBalancer?: LoadBalancerConfig
  rateLimit?: RateLimitConfig
  certificates?: Certificate[]
  sslCertificates?: SSLCertificate[]
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

export type Site = {
  id: string
  name: string
  hostname: string
  ip: string
  created_at?: string
  updated_at?: string
}

export type SiteCreateRequest = {
  name: string
  hostname: string
  ip: string
}

export type SiteUpdateRequest = {
  name?: string
  hostname?: string
  ip?: string
}

export type SiteListResponse = {
  items: Site[]
}

export type SiteSummary = {
  id: string
  name: string
  hostname: string
  ip: string
}

export type SiteGroup = {
  group_key: string
  count: number
  sites: SiteSummary[]
}

export type SiteGroupResponse = {
  groups: SiteGroup[]
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
  status: string
  version_id: string
}

export type SiteLogStream = {
  site_id: string
  filter_query: string
  updated_at?: string
}

export type SiteLogKind = 'all' | 'traffic' | 'system'

export type SiteLogEntry = {
  id: string
  site_id: string
  kind: SiteLogKind
  level: string
  message: string
  request_id?: string
  client_ip?: string
  host?: string
  method?: string
  path?: string
  status_code?: number
  latency_ms?: number
  operator?: string
  action?: string
  detail?: string
  created_at: string
}

export type SiteLogHistoryResponse = {
  items: SiteLogEntry[]
}

export type ConfigValidationResponse = {
  valid: boolean
  errors: ValidationIssue[]
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
