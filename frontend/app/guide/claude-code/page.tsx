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
import { Badge } from "@/components/ui/badge"
import { Copy, Check, Terminal } from "lucide-react"

export default function ClaudeCodeGuidePage() {
  const [copiedStep, setCopiedStep] = useState<number | null>(null)

  const copy = (text: string, step: number) => {
    navigator.clipboard.writeText(text)
    setCopiedStep(step)
    setTimeout(() => setCopiedStep(null), 2000)
  }

  const steps = [
    {
      title: "Set the base URL",
      description: "Tell Claude Code to route requests through the proxy instead of directly to Anthropic.",
      command: "export ANTHROPIC_BASE_URL=http://127.0.0.1:8000",
    },
    {
      title: "Set the API key (if auth guard is enabled)",
      description: "If the proxy has an API key configured, set it as the Anthropic API key.",
      command: "export ANTHROPIC_API_KEY=your-proxy-api-key",
    },
    {
      title: "Start Claude Code",
      description: "Launch Claude Code as normal. It will now route through the Kiro proxy.",
      command: "claude",
    },
    {
      title: "Verify the connection",
      description: "Run a simple prompt to verify the proxy is working.",
      command: 'claude -p "Hello, what model are you?"',
    },
  ]

  return (
    <div>
      <Header
        title="Claude Code Setup"
        description="Connect Anthropic's CLI tool to the Kiro proxy"
      />
      <div className="p-6 space-y-6">
        {/* Overview */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Terminal className="h-8 w-8 text-primary" />
              <div>
                <CardTitle>Claude Code</CardTitle>
                <CardDescription>
                  Anthropic&apos;s official CLI tool for Claude
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Claude Code uses the <code className="font-mono text-xs bg-muted px-1 rounded">ANTHROPIC_BASE_URL</code> environment
              variable to determine where to send API requests. By pointing it at the
              Kiro proxy, all requests will be transparently routed through Kiro upstream.
            </p>
          </CardContent>
        </Card>

        {/* Steps */}
        <div className="space-y-4">
          {steps.map((step, i) => (
            <Card key={i}>
              <CardHeader className="pb-3">
                <div className="flex items-center gap-3">
                  <Badge variant="outline">{i + 1}</Badge>
                  <CardTitle className="text-base">{step.title}</CardTitle>
                </div>
                <CardDescription>{step.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded bg-muted px-4 py-2 text-sm font-mono">
                    {step.command}
                  </code>
                  <button
                    onClick={() => copy(step.command, i)}
                    className="rounded-md p-2 hover:bg-accent transition-colors"
                    aria-label="Copy command"
                  >
                    {copiedStep === i ? (
                      <Check className="h-4 w-4 text-green-600" />
                    ) : (
                      <Copy className="h-4 w-4 text-muted-foreground" />
                    )}
                  </button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Troubleshooting */}
        <Card>
          <CardHeader>
            <CardTitle>Troubleshooting</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="font-medium text-sm">Connection refused</p>
              <p className="text-sm text-muted-foreground">
                Make sure the proxy is running: <code className="font-mono text-xs bg-muted px-1 rounded">go run ./cmd/server</code>
              </p>
            </div>
            <div>
              <p className="font-medium text-sm">Authentication error</p>
              <p className="text-sm text-muted-foreground">
                Check that your Kiro credentials are valid and not expired. The proxy logs will show the auth status.
              </p>
            </div>
            <div>
              <p className="font-medium text-sm">Model not found</p>
              <p className="text-sm text-muted-foreground">
                The proxy passes unknown model names to Kiro. If Kiro doesn&apos;t support the model, you&apos;ll get an error from upstream.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
