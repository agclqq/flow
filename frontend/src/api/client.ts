export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const API_BASE = '/api/v1'

// 统一响应封装：{"code":0,"msg":"","data":{}}
type Envelope<T> = {
  code: number
  msg?: string
  data?: T
}

async function parseErrorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as Envelope<unknown> & { error?: string; message?: string }
    return body.msg ?? body.error ?? body.message ?? res.statusText
  } catch {
    return res.statusText
  }
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  })

  if (!res.ok) {
    // 非 200 仅出现在网关/路由层异常，尽量解析统一格式中的 msg。
    const message = await parseErrorMessage(res)
    throw new ApiError(res.status, message)
  }

  const text = await res.text()
  if (!text) {
    return undefined as T
  }

  const body = JSON.parse(text) as Envelope<T>
  // 业务错误码非 0：抛出错误，msg 为可展示信息。
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '请求失败')
  }
  return body.data as T
}

export function buildQuery(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      search.set(key, String(value))
    }
  }
  const query = search.toString()
  return query ? `?${query}` : ''
}
