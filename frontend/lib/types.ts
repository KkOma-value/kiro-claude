// Types matching Go backend API responses

export interface StatusResponse {
  mode: string
  bind_address: string
  auth_guard: boolean
  auth_guard_masked_key?: string
  debug_dump: boolean
  accounts: AccountInfo[]
  upstream_endpoint: string
  uptime_seconds: number
  version: string
}

export interface AccountInfo {
  id: string
  auth_type: string
  healthy: boolean
  failures: number
  is_current: boolean
}

export interface ResolveRequest {
  model: string
}

export interface ResolveResponse {
  original: string
  normalized: string
  internal_id: string
  source: string
  pipeline: PipelineStep[]
}

export interface PipelineStep {
  step: string
  input: string
  output: string
}

export interface EndpointsResponse {
  endpoints: EndpointInfo[]
}

export interface EndpointInfo {
  path: string
  method: string
  status: string
  latency_ms: number
}

export interface ModelsResponse {
  data: ModelData[]
  object: string
}

export interface ModelData {
  id: string
  object: string
  owned_by: string
}

export interface MessageRequest {
  model: string
  messages: { role: string; content: string }[]
  max_tokens?: number
  stream?: boolean
}

export interface MessageResponse {
  id: string
  type: string
  role: string
  content: { type: string; text?: string; name?: string }[]
  model: string
  stop_reason: string
  usage: { input_tokens: number; output_tokens: number }
}
