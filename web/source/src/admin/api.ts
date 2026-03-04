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
  LoginCaptchaRequirementResponse,
  LoginPayload,
  LoginResponse,
} from './types'

const API_BASE = '/admin/api/v1'

async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
  token?: string | null,
) {
  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body) {
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
    const message = error.message || `请求失败(${response.status})`
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
      blockPageHtml: config?.security?.blockPageHtml ?? '',
      enableHsts: config?.security?.enableHsts ?? false,
    },
    routing: {
      defaultUpstream: config?.routing?.defaultUpstream ?? '',
      domains: config?.routing?.domains ?? [],
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
