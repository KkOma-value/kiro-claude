import type {
  StatusResponse,
  ResolveResponse,
  EndpointsResponse,
  ModelsResponse,
  MessageRequest,
  MessageResponse,
} from './types'

const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:8000'

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${BACKEND_URL}${path}`
  const res = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`API error ${res.status}: ${text}`)
  }
  return res.json()
}

export const api = {
  getStatus: () => fetchJSON<StatusResponse>('/api/status'),

  getModels: () => fetchJSON<ModelsResponse>('/v1/models'),

  resolveModel: (model: string) =>
    fetchJSON<ResolveResponse>('/api/resolve-model', {
      method: 'POST',
      body: JSON.stringify({ model }),
    }),

  getEndpoints: () => fetchJSON<EndpointsResponse>('/api/endpoints'),

  sendMessage: (req: MessageRequest) =>
    fetchJSON<MessageResponse>('/v1/messages', {
      method: 'POST',
      body: JSON.stringify(req),
    }),
}
