"use client"

import { useState } from "react"
import { Header } from "@/components/layout/header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Send, Copy, Check } from "lucide-react"
import type { MessageResponse } from "@/lib/types"

export default function ApiTestPage() {
  const [model, setModel] = useState("claude-sonnet-4.5")
  const [message, setMessage] = useState("Hello, what model are you?")
  const [response, setResponse] = useState<MessageResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const handleSend = async () => {
    setLoading(true)
    setError(null)
    setResponse(null)

    try {
      const res = await fetch("/api/test-message", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          model,
          messages: [{ role: "user", content: message }],
          max_tokens: 1024,
          stream: false,
        }),
      })

      if (!res.ok) {
        const text = await res.text()
        throw new Error(`HTTP ${res.status}: ${text}`)
      }

      const data = await res.json()
      setResponse(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed")
    } finally {
      setLoading(false)
    }
  }

  const copyResponse = () => {
    if (response) {
      navigator.clipboard.writeText(JSON.stringify(response, null, 2))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div>
      <Header
        title="API Tester"
        description="Send test requests to the proxy"
      />
      <div className="p-6 space-y-6">
        {/* Request Builder */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Send className="h-5 w-5" />
              Request Builder
            </CardTitle>
            <CardDescription>
              Construct and send a message to POST /v1/messages
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Model</label>
              <Input
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="claude-sonnet-4.5"
                className="font-mono text-sm"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Message</label>
              <textarea
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Enter your message..."
                className="w-full rounded-md border bg-background px-3 py-2 text-sm min-h-[100px] resize-y"
              />
            </div>
            <Button onClick={handleSend} disabled={loading}>
              {loading ? "Sending..." : "Send Request"}
            </Button>
          </CardContent>
        </Card>

        {/* Error */}
        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <p className="text-sm text-destructive">{error}</p>
            </CardContent>
          </Card>
        )}

        {/* Response */}
        {response && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Response</CardTitle>
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{response.model}</Badge>
                  <Badge>{response.stop_reason}</Badge>
                  <button
                    onClick={copyResponse}
                    className="rounded-md p-1.5 hover:bg-accent transition-colors"
                  >
                    {copied ? (
                      <Check className="h-4 w-4 text-green-600" />
                    ) : (
                      <Copy className="h-4 w-4 text-muted-foreground" />
                    )}
                  </button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Content */}
              <div className="rounded-md border p-4 bg-muted/30">
                {response.content.map((block, i) => (
                  <div key={i}>
                    {block.type === "text" && (
                      <p className="text-sm whitespace-pre-wrap">{block.text}</p>
                    )}
                    {block.type === "tool_use" && (
                      <div className="rounded border bg-card p-2 mt-2">
                        <p className="text-xs font-medium">Tool: {block.name}</p>
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {/* Usage */}
              <div className="flex gap-4 text-xs text-muted-foreground">
                <span>Input tokens: {response.usage.input_tokens}</span>
                <span>Output tokens: {response.usage.output_tokens}</span>
                <span>ID: {response.id}</span>
              </div>

              {/* Raw JSON */}
              <details className="text-xs">
                <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
                  Raw JSON
                </summary>
                <pre className="mt-2 rounded bg-muted p-4 overflow-auto max-h-[300px] font-mono">
                  {JSON.stringify(response, null, 2)}
                </pre>
              </details>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
