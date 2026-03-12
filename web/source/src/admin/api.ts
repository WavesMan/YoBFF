import type {
  ApiError,
  CaptchaResponse,
  CdnConfigResponse,
  Config,
  ConfigRollbackPayload,
  ConfigRollbackResponse,
  ConfigValidationResponse,
  ConfigVersionResponse,
  HealthzResponse,
  LogLevelResponse,
  LogStats,
  LBPool,
  LBRouteRule,
  LoginCaptchaRequirementResponse,
  LoginPayload,
  LoginResponse,
  Site,
  SiteCreateRequest,
  SiteGroupResponse,
  SiteListResponse,
  SiteLogHistoryResponse,
  SiteLogKind,
  SiteLogStream,
  SiteUpdateRequest,
  SiteCDNOriginRefreshRequest,
  SiteCDNOriginRefreshResponse,
  SiteCDNOriginStatusListResponse,
  SSLCertificate,
} from './types'

const API_BASE = '/admin/api/v1'

async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
  token?: string | null,
) {
  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const response = await fetch(path, {
    ...options,
    headers,
  })
  const contentType = response.headers.get('content-type') || ''
  const isJson = contentType.includes('application/json')
  const data = isJson ? await response.json() : null
  if (!response.ok) {
    const error = (data || {}) as ApiError
    const issues = (error.errors || [])
      .map(issue => `${issue.path}: ${issue.message}`)
      .join('; ')
    const detailText = issues ? `，详情：${issues}` : ''
    const requestIDText = error.request_id ? ` [request_id=${error.request_id}]` : ''
    const errorCodeText = error.error_code ? ` [error_code=${error.error_code}]` : ''
    const message = (error.message || `请求失败(${response.status})`) + detailText + errorCodeText + requestIDText
    throw new Error(message)
  }
  return data as T
}

export function normalizeConfig(config?: Config | null): Config {
  return {
    dataPlane: {
      httpListenAddr: config?.dataPlane?.httpListenAddr ?? '',
      httpsListenAddr: config?.dataPlane?.httpsListenAddr ?? '',
      enableHttps: config?.dataPlane?.enableHttps ?? false,
    },
    controlPlane: {
      adminListenAddr: config?.controlPlane?.adminListenAddr ?? '',
      auth: {
        username: config?.controlPlane?.auth?.username ?? '',
        password: config?.controlPlane?.auth?.password ?? '',
        token: config?.controlPlane?.auth?.token ?? '',
      },
    },
    security: {
      allowedCidrs: config?.security?.allowedCidrs ?? [],
      trustedProxyCidrs: config?.security?.trustedProxyCidrs ?? [],
      allowedCdnProviders: config?.security?.allowedCdnProviders ?? [],
      cdnProviderSettings: config?.security?.cdnProviderSettings ?? {},
      originProtectionMode: config?.security?.originProtectionMode,
      enableHsts: config?.security?.enableHsts ?? false,
    },
    routing: {
      defaultUpstream: config?.routing?.defaultUpstream ?? '',
      domains: config?.routing?.domains ?? [],
    },
    loadBalancer: {
      defaultPoolId: config?.loadBalancer?.defaultPoolId ?? '',
      pools: config?.loadBalancer?.pools ?? [],
      routes: config?.loadBalancer?.routes ?? [],
    },
    certificates: config?.certificates ?? [],
    cdnSync: {
      enabled: config?.cdnSync?.enabled ?? false,
      providers: config?.cdnSync?.providers ?? [],
      schedule: config?.cdnSync?.schedule ?? '',
      cloudflare: {
        ipv4_url: config?.cdnSync?.cloudflare?.ipv4_url ?? '',
        ipv6_url: config?.cdnSync?.cloudflare?.ipv6_url ?? '',
        endpoint: config?.cdnSync?.cloudflare?.endpoint ?? '',
        api_token: config?.cdnSync?.cloudflare?.api_token ?? '',
        zone_id: config?.cdnSync?.cloudflare?.zone_id ?? '',
      },
    },
  }
}

export async function fetchCertificates(
  token: string,
  page = 1,
  pageSize = 15
) {
  return apiRequest<{ items: SSLCertificate[], total: number }>(
    `${API_BASE}/certs?page=${page}&pageSize=${pageSize}`,
    { method: 'GET' },
    token
  )
}

export async function uploadCertificate(
  token: string,
  formData: FormData
) {
  return apiRequest<SSLCertificate>(`${API_BASE}/certs`, {
    method: 'POST',
    body: formData,
  }, token)
}

export async function deleteCertificate(
  token: string,
  id: string
) {
  return apiRequest(
    `${API_BASE}/certs/${id}`,
    { method: 'DELETE' },
    token
  )
}

export async function fetchHealthz() {
  return apiRequest<HealthzResponse>('/healthz')
}

export async function fetchLogStats(token: string) {
  return apiRequest<LogStats>(`${API_BASE}/log/stats`, {}, token)
}

export async function fetchLogLevel(token: string) {
  return apiRequest<LogLevelResponse>(`${API_BASE}/log/level`, {}, token)
}

export async function updateLogLevel(token: string, level: string) {
  return apiRequest<LogLevelResponse>(`${API_BASE}/log/level`, {
    method: 'PUT',
    body: JSON.stringify({ level }),
  }, token)
}

export async function fetchConfig(token: string) {
  return apiRequest<Config>(`${API_BASE}/config`, {}, token)
}

export async function applyConfig(token: string, config: Config, operator: string) {
  return apiRequest(`${API_BASE}/config`, {
    method: 'PUT',
    body: JSON.stringify(config),
    headers: {
      'X-Operator': operator,
    },
  }, token)
}

export async function reloadConfig(token: string) {
  return apiRequest(`${API_BASE}/config/reload`, { method: 'POST' }, token)
}

export async function validateConfig(token: string, config: Config, operator: string) {
  return apiRequest<ConfigValidationResponse>(`${API_BASE}/config/validate`, {
    method: 'POST',
    body: JSON.stringify(config),
    headers: {
      'X-Operator': operator,
    },
  }, token)
}

export async function fetchConfigVersions(token: string, limit = 20) {
  return apiRequest<ConfigVersionResponse>(`${API_BASE}/config/versions?limit=${limit}`, {}, token)
}

export async function rollbackConfig(token: string, payload: ConfigRollbackPayload, operator: string) {
  return apiRequest<ConfigRollbackResponse>(`${API_BASE}/config/rollback`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'X-Operator': operator,
    },
  }, token)
}

export async function fetchCdnConfig(token: string) {
  return apiRequest<CdnConfigResponse>(`${API_BASE}/config/cdn`, {}, token)
}

export async function fetchCaptcha(preset: 'easy' | 'default' = 'default') {
  return apiRequest<CaptchaResponse>(`${API_BASE}/captcha?preset=${preset}`)
}

export async function loginAdmin(payload: LoginPayload) {
  return apiRequest<LoginResponse>(`${API_BASE}/login`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function fetchLoginCaptchaRequirement() {
  return apiRequest<LoginCaptchaRequirementResponse>(`${API_BASE}/login/require-captcha`)
}

export async function logoutAdmin(token: string) {
  return apiRequest(`${API_BASE}/logout`, { method: 'POST' }, token)
}

export async function fetchSites(token: string, query?: { hostname?: string; ip?: string }) {
  const params = new URLSearchParams()
  if (query?.hostname) params.set('hostname', query.hostname)
  if (query?.ip) params.set('ip', query.ip)
  return apiRequest<SiteListResponse>(`${API_BASE}/sites?${params.toString()}`, {}, token)
}

export async function createSite(token: string, data: SiteCreateRequest) {
  return apiRequest<Site>(`${API_BASE}/sites`, {
    method: 'POST',
    body: JSON.stringify(data),
  }, token)
}

export async function fetchSite(token: string, id: string) {
  return apiRequest<Site>(`${API_BASE}/sites/${id}`, {}, token)
}

export async function updateSite(token: string, id: string, data: SiteUpdateRequest) {
  return apiRequest<Site>(`${API_BASE}/sites/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }, token)
}

export async function deleteSite(token: string, id: string) {
  return apiRequest(`${API_BASE}/sites/${id}`, {
    method: 'DELETE',
  }, token)
}

export async function fetchSiteConfig(token: string, id: string) {
  return apiRequest<Config>(`${API_BASE}/sites/${id}/config`, {}, token)
}

export async function updateSiteConfig(token: string, id: string, config: Config, operator: string) {
  return apiRequest<{ status: string }>(`${API_BASE}/sites/${id}/config`, {
    method: 'PUT',
    body: JSON.stringify(config),
    headers: { 'X-Operator': operator },
  }, token)
}

export async function fetchSiteCDNOriginStatus(token: string, id: string) {
  try {
    return await apiRequest<SiteCDNOriginStatusListResponse>(`${API_BASE}/sites/${id}/cdn/origin/status`, {}, token)
  } catch (err) {
    const message = err instanceof Error ? err.message : ''
    if (message.includes('[error_code=not_found]') && API_BASE.startsWith('/admin')) {
      const altBase = API_BASE.replace(/^\/admin/, '')
      return apiRequest<SiteCDNOriginStatusListResponse>(`${altBase}/sites/${id}/cdn/origin/status`, {}, token)
    }
    throw err
  }
}

export async function refreshSiteCDNOrigin(
  token: string,
  id: string,
  payload: SiteCDNOriginRefreshRequest,
  operator: string
) {
  try {
    return await apiRequest<SiteCDNOriginRefreshResponse>(`${API_BASE}/sites/${id}/cdn/origin/refresh`, {
      method: 'POST',
      body: JSON.stringify(payload ?? {}),
      headers: { 'X-Operator': operator },
    }, token)
  } catch (err) {
    const message = err instanceof Error ? err.message : ''
    if (message.includes('[error_code=not_found]') && API_BASE.startsWith('/admin')) {
      const altBase = API_BASE.replace(/^\/admin/, '')
      return apiRequest<SiteCDNOriginRefreshResponse>(`${altBase}/sites/${id}/cdn/origin/refresh`, {
        method: 'POST',
        body: JSON.stringify(payload ?? {}),
        headers: { 'X-Operator': operator },
      }, token)
    }
    throw err
  }
}

export async function fetchSiteVersions(token: string, id: string, limit = 20) {
  return apiRequest<ConfigVersionResponse>(`${API_BASE}/sites/${id}/config/versions?limit=${limit}`, {}, token)
}

export async function rollbackSiteConfig(token: string, id: string, versionId: string, operator: string) {
  return apiRequest<ConfigRollbackResponse>(`${API_BASE}/sites/${id}/config/rollback`, {
    method: 'POST',
    body: JSON.stringify({ version_id: versionId }),
    headers: { 'X-Operator': operator },
  }, token)
}

export async function deleteSiteVersion(token: string, id: string, versionId: string) {
  return apiRequest<{ status: string; version_id: string }>(`${API_BASE}/sites/${id}/config/versions/${encodeURIComponent(versionId)}`, {
    method: 'DELETE',
  }, token)
}

export async function fetchSiteGroups(token: string, by: 'hostname' | 'ip') {
  return apiRequest<SiteGroupResponse>(`${API_BASE}/site-groups?by=${by}`, {}, token)
}

export async function fetchSiteLogStream(token: string, id: string) {
  return apiRequest<SiteLogStream>(`${API_BASE}/sites/${id}/log/stream`, {}, token)
}

export async function updateSiteLogStream(token: string, id: string, filterQuery: string) {
  return apiRequest<SiteLogStream>(`${API_BASE}/sites/${id}/log/stream`, {
    method: 'PUT',
    body: JSON.stringify({ site_id: id, filter_query: filterQuery }),
  }, token)
}

export async function fetchSiteLogHistory(
  token: string,
  id: string,
  params: {
    kind: SiteLogKind
    level: string
    startTime: string
    endTime: string
    limit: number
  }
) {
  const search = new URLSearchParams()
  if (params.kind) search.set('kind', params.kind)
  if (params.level) search.set('level', params.level)
  if (params.startTime) search.set('start_time', params.startTime)
  if (params.endTime) search.set('end_time', params.endTime)
  if (params.limit > 0) search.set('limit', String(params.limit))
  return apiRequest<SiteLogHistoryResponse>(`${API_BASE}/sites/${id}/log/history?${search.toString()}`, {}, token)
}

export async function fetchLBPools(token: string) {
  return apiRequest<{ items: LBPool[] }>(`${API_BASE}/lb/pools`, { method: 'GET' }, token)
}

export async function createLBPool(token: string, pool: LBPool) {
  return apiRequest<{ status: string; id: string }>(`${API_BASE}/lb/pools`, {
    method: 'POST',
    body: JSON.stringify(pool),
  }, token)
}

export async function updateLBPool(token: string, poolID: string, pool: LBPool) {
  return apiRequest<{ status: string; pool_id: string }>(`${API_BASE}/lb/pools/${encodeURIComponent(poolID)}`, {
    method: 'PUT',
    body: JSON.stringify(pool),
  }, token)
}

export async function deleteLBPool(token: string, poolID: string) {
  return apiRequest<{ status: string; pool_id: string }>(`${API_BASE}/lb/pools/${encodeURIComponent(poolID)}`, {
    method: 'DELETE',
  }, token)
}

export async function fetchLBRoutes(token: string) {
  return apiRequest<{ items: LBRouteRule[]; default_pool_id?: string }>(`${API_BASE}/lb/routes`, { method: 'GET' }, token)
}

export async function upsertLBRoute(token: string, domain: string, route: LBRouteRule) {
  return apiRequest<{ status: string; domain: string }>(`${API_BASE}/lb/routes/${encodeURIComponent(domain)}`, {
    method: 'PUT',
    body: JSON.stringify(route),
  }, token)
}

export async function deleteLBRoute(token: string, domain: string) {
  return apiRequest<{ status: string; domain: string }>(`${API_BASE}/lb/routes/${encodeURIComponent(domain)}`, {
    method: 'DELETE',
  }, token)
}
