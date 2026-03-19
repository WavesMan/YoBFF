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
  accountId?: string
  zoneId?: string
  endpoint?: string
  refreshIntervalSeconds?: number
  maxStalenessSeconds?: number
}

export type SecurityConfig = {
  allowedCidrs?: string[]
  trustedProxyCidrs?: string[]
  allowedCdnProviders?: string[]
  cdnProviderSettings?: Record<string, CDNProviderSetting>
  originProtectionMode?: 'disabled' | 'enforced'
  enableHsts?: boolean
}

export type Certificate = {
  domain?: string
  certFile?: string
  keyFile?: string
  certPem?: string
  keyPem?: string
}

export type CDNSyncConfig = {
  enabled?: boolean
  providers?: string[]
  schedule?: string
  cloudflare?: {
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

export type AuditLogEntry = {
  id: string
  action: string
  target?: string
  created_at: string
  operator?: string
  detail?: string
}

export type AuditLogListResponse = {
  items: AuditLogEntry[]
  total?: number
  page?: number
  pageSize?: number
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
  operator?: string
}

export type LoginCaptchaRequirementResponse = {
  required: boolean
}

export type SiteCDNOriginStatus = {
  site_id: string
  provider: string
  last_attempt_at?: string
  last_success_at?: string
  consecutive_failures: number
  last_error?: string
  updated_at: string
}

export type SiteCDNOriginStatusListResponse = {
  items: SiteCDNOriginStatus[]
}

export type SiteCDNOriginRefreshRequest = {
  providers?: string[]
}

export type SiteCDNOriginRefreshResult = {
  provider: string
  ok: boolean
  count?: number
  error?: string
}

export type SiteCDNOriginRefreshResponse = {
  status: string
  results: SiteCDNOriginRefreshResult[]
}

export type WeaverInput = {
  name: string
  payload: Record<string, unknown>
}

export type WeaverDraft = {
  id: string
  name: string
  inputs: WeaverInput[]
  mapping: Record<string, unknown>
  dag?: WeaverDAG
  created_at: string
  updated_at: string
  operator?: string
}

export type WeaverDraftListResponse = {
  items: WeaverDraft[]
}

export type WeaverRunSource = {
  name: string
  ok: boolean
  code?: string
  error?: string
}

export type WeaverRunNodeExecution = {
  node_id: string
  node_type: string
  attempt: number
  status: 'succeeded' | 'failed'
  duration_ms: number
  input_keys: string[]
  parent_nodes: string[]
  output_keys: string[]
  error?: string
}

export type WeaverRunFailure = {
  scope: 'source' | 'node' | 'dag'
  code: string
  node_id?: string
  source?: string
  attempt: number
  error: string
  retryable: boolean
}

export type WeaverRunAttempt = {
  attempt: number
  status: 'succeeded' | 'failed'
  duration_ms: number
  node_executions: WeaverRunNodeExecution[]
  failure?: WeaverRunFailure
}

export type WeaverRunRetrySnapshot = {
  max_attempts: number
  used: number
  triggered: boolean
  retryable_codes: string[]
  backoff_initial_ms: number
  backoff_multiplier: number
  backoff_max_ms: number
  delays_ms: number[]
}

export type WeaverRunRetryPolicy = {
  max_attempts?: number
  retry_on_node_error?: boolean
  retry_on_source_error?: boolean
  retryable_codes?: string[]
  backoff_initial_ms?: number
  backoff_multiplier?: number
  backoff_max_ms?: number
}

export type WeaverRunResponse = {
  run_id: string
  status: 'succeeded' | 'failed'
  output: unknown
  duration_ms: number
  sources: WeaverRunSource[]
  attempts: WeaverRunAttempt[]
  failures: WeaverRunFailure[]
  retry: WeaverRunRetrySnapshot
}

export type WeaverRunErrorGroup = {
  code: string
  count: number
}

export type WeaverRunTrendPoint = {
  run_id: string
  created_at: string
  status: 'succeeded' | 'failed'
  duration_ms: number
  attempts_used: number
  error_codes: string[]
}

export type WeaverRunStatsResponse = {
  limit: number
  scope?: 'draft' | 'version'
  target_id?: string
  total_runs: number
  success_runs: number
  failed_runs: number
  error_groups: WeaverRunErrorGroup[]
  trends: WeaverRunTrendPoint[]
}

export type WeaverDeleteResponse = {
  status: 'deleted'
  id: string
}

export type WeaverDAGNode = {
  id: string
  type: string
  inputs?: string[]
  outputs?: string[]
  config?: Record<string, unknown>
}

export type WeaverDAGEdge = {
  from: string
  to: string
}

export type WeaverDAG = {
  nodes: WeaverDAGNode[]
  edges: WeaverDAGEdge[]
  output_node_id?: string
}

export type WeaverNodeContract = {
  node_id: string
  type: string
  inputs: string[]
  outputs: string[]
}

export type WeaverNodeContractCatalogResponse = {
  items: WeaverNodeContract[]
}

export type WeaverDraftValidateResponse = {
  status: 'valid'
  draft_id: string
  node_contracts: WeaverNodeContract[]
}

export type WeaverVersion = {
  id: string
  draft_id: string
  version: number
  name: string
  inputs: WeaverInput[]
  mapping: Record<string, unknown>
  dag: WeaverDAG
  node_contracts: WeaverNodeContract[]
  created_at: string
  operator?: string
  source?: string
}

export type WeaverVersionListResponse = {
  items: WeaverVersion[]
}
