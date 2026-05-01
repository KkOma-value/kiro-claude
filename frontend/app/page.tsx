"use client"

import { useEffect, useState } from "react"
import { Header } from "@/components/layout/header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Activity,
  Shield,
  Server,
  Clock,
  ArrowRight,
  Copy,
  Check,
} from "lucide-react"
import type { StatusResponse } from "@/lib/types"

export default function DashboardPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    fetch("/api/status")
      .then((res) => res.json())
      .then(setStatus)
      .catch((err) => setError(err.message))
  }, [])

  const copyCommand = () => {
    const addr = status?.bind_address || "127.0.0.1:8000"
    navigator.clipboard.writeText(
      `export ANTHROPIC_BASE_URL=http://${addr}`
    )
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div>
      <Header title="Dashboard" description="Proxy overview and quick start" />
      <div className="p-6 space-y-6">
        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <p className="text-sm text-destructive">
                Failed to connect to backend: {error}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Make sure the Go backend is running on port 8000.
              </p>
            </CardContent>
          </Card>
        )}

        {/* Status Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Runtime Mode</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {status?.mode || "..."}
              </div>
              <p className="text-xs text-muted-foreground">
                {status?.upstream_endpoint
                  ? new URL(status.upstream_endpoint).hostname
                  : "Not connected"}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Auth Guard</CardTitle>
              <Shield className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {status?.auth_guard ? (
                  <Badge variant="default" className="bg-green-600">Enabled</Badge>
                ) : (
                  <Badge variant="secondary">Disabled</Badge>
                )}
              </div>
              {status?.auth_guard_masked_key && (
                <p className="mt-1 text-xs text-muted-foreground font-mono">
                  Key: {status.auth_guard_masked_key}
                </p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Accounts</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {status?.accounts.length || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {status?.accounts.filter((a) => a.healthy).length || 0} healthy
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Uptime</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {status ? formatUptime(status.uptime_seconds) : "..."}
              </div>
              <p className="text-xs text-muted-foreground">
                v{status?.version || "?"}
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Architecture Diagram */}
        <Card>
          <CardHeader>
            <CardTitle>Architecture</CardTitle>
            <CardDescription>
              How the proxy connects Claude Code to Kiro upstream
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-center gap-4 py-8 text-sm">
              <div className="rounded-lg border bg-card p-4 text-center shadow-sm">
                <div className="font-semibold">Claude Code</div>
                <div className="text-xs text-muted-foreground">
                  / Cursor / Cline
                </div>
              </div>
              <ArrowRight className="h-5 w-5 text-muted-foreground" />
              <div className="rounded-lg border-2 border-primary bg-primary/5 p-4 text-center shadow-sm">
                <div className="font-semibold text-primary">Kiro Proxy</div>
                <div className="text-xs text-muted-foreground">
                  {status?.bind_address || "127.0.0.1:8000"}
                </div>
              </div>
              <ArrowRight className="h-5 w-5 text-muted-foreground" />
              <div className="rounded-lg border bg-card p-4 text-center shadow-sm">
                <div className="font-semibold">Kiro Upstream</div>
                <div className="text-xs text-muted-foreground">
                  CodeWhisperer API
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Quick Start */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Start</CardTitle>
            <CardDescription>
              Connect Claude Code to this proxy in one command
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded bg-muted px-4 py-3 text-sm font-mono">
                export ANTHROPIC_BASE_URL=http://{status?.bind_address || "127.0.0.1:8000"}
              </code>
              <button
                onClick={copyCommand}
                className="rounded-md p-2 hover:bg-accent transition-colors"
                aria-label="Copy command"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground" />
                )}
              </button>
            </div>
            <p className="text-sm text-muted-foreground">
              Set this environment variable before starting Claude Code. The proxy
              will intercept all Anthropic API calls and route them through Kiro.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}
