import { NextRequest, NextResponse } from "next/server"
import { BACKEND_URL } from "@/lib/constants"

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const res = await fetch(`${BACKEND_URL}/api/resolve-model`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
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
