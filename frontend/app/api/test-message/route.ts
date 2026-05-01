import { NextRequest, NextResponse } from "next/server"
import {
  BACKEND_URL,
  MODEL_NAME_PATTERN,
  MAX_MESSAGE_CONTENT_LENGTH,
  MAX_MESSAGES_COUNT,
} from "@/lib/constants"

/**
 * Simple in-memory rate limiter.
 * Allows at most MAX_REQUESTS requests per WINDOW_MS window.
 */
const WINDOW_MS = 60_000 // 1 minute
const MAX_REQUESTS = 10

let requestTimestamps: number[] = []

function isRateLimited(): boolean {
  const now = Date.now()
  requestTimestamps = requestTimestamps.filter((t) => now - t < WINDOW_MS)
  if (requestTimestamps.length >= MAX_REQUESTS) {
    return true
  }
  requestTimestamps.push(now)
  return false
}

export async function POST(request: NextRequest) {
  // Rate limiting
  if (isRateLimited()) {
    return NextResponse.json(
      { error: "Rate limit exceeded. Max 10 requests per minute." },
      { status: 429 }
    )
  }

  try {
    // Validate Content-Length early to reject oversized payloads
    const contentLength = request.headers.get("content-length")
    if (contentLength && parseInt(contentLength, 10) > 32 * 1024) {
      return NextResponse.json(
        { error: "Request body too large. Max 32 KB." },
        { status: 413 }
      )
    }

    const body = await request.json()

    // --- Input validation ---

    // model: required, must match safe pattern
    if (!body.model || typeof body.model !== "string") {
      return NextResponse.json(
        { error: "model field is required and must be a string" },
        { status: 400 }
      )
    }
    if (!MODEL_NAME_PATTERN.test(body.model)) {
      return NextResponse.json(
        { error: "Invalid model name. Only alphanumeric, dot, hyphen, underscore allowed (max 128 chars)." },
        { status: 400 }
      )
    }

    // messages: required, must be a non-empty array
    if (!Array.isArray(body.messages) || body.messages.length === 0) {
      return NextResponse.json(
        { error: "messages must be a non-empty array" },
        { status: 400 }
      )
    }

    if (body.messages.length > MAX_MESSAGES_COUNT) {
      return NextResponse.json(
        { error: `Too many messages. Max ${MAX_MESSAGES_COUNT} allowed in test mode.` },
        { status: 400 }
      )
    }

    // Validate each message
    for (const msg of body.messages) {
      if (!msg.role || typeof msg.role !== "string") {
        return NextResponse.json(
          { error: "Each message must have a role string" },
          { status: 400 }
        )
      }
      if (!["user", "assistant"].includes(msg.role)) {
        return NextResponse.json(
          { error: `Invalid role "${msg.role}". Only "user" and "assistant" allowed.` },
          { status: 400 }
        )
      }
      if (!msg.content || typeof msg.content !== "string") {
        return NextResponse.json(
          { error: "Each message must have a content string" },
          { status: 400 }
        )
      }
      if (msg.content.length > MAX_MESSAGE_CONTENT_LENGTH) {
        return NextResponse.json(
          { error: `Message content too long. Max ${MAX_MESSAGE_CONTENT_LENGTH} characters per message.` },
          { status: 400 }
        )
      }
    }

    // Build sanitized request — only forward safe fields
    const sanitizedBody = {
      model: body.model,
      messages: body.messages.map((m: { role: string; content: string }) => ({
        role: m.role,
        content: m.content,
      })),
      max_tokens: Math.min(Number(body.max_tokens) || 1024, 4096),
      stream: false, // Always disable streaming in test mode
    }

    const res = await fetch(`${BACKEND_URL}/v1/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(sanitizedBody),
    })

    if (!res.ok) {
      const text = await res.text()
      return NextResponse.json(
        { error: `Backend returned ${res.status}: ${text}` },
        { status: res.status }
      )
    }

    const data = await res.json()
    return NextResponse.json(data)
  } catch (err) {
    return NextResponse.json(
      { error: `Failed to connect to backend: ${err instanceof Error ? err.message : "unknown"}` },
      { status: 502 }
    )
  }
}
