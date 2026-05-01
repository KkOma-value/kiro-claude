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
import { Settings, Globe, Shield, Wrench } from "lucide-react"
import type { StatusResponse } from "@/lib/types"

export default function ConfigPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null)

  useEffect(() => {
    fetch("/api/status")
      .then((res) => res.json())
      .then(setStatus)
      .catch(() => {})
  }, [])

  return (
    <div>
      <Header
        title="Configuration"
        description="Current proxy configuration (read-only)"
      />
      <div className="p-6 space-y-6">
        {/* Runtime Config */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Runtime
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <ConfigRow label="Mode" value={status?.mode || "..."} />
            <ConfigRow label="Bind Address" value={status?.bind_address || "..."} mono />
            <ConfigRow label="Version" value={status?.version || "..."} />
            <ConfigRow
              label="Uptime"
              value={status ? `${status.uptime_seconds}s` : "..."}
            />
          </CardContent>
        </Card>

        {/* Upstream Config */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Globe className="h-5 w-5" />
              Upstream
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ConfigRow
              label="Endpoint"
              value={status?.upstream_endpoint || "..."}
              mono
            />
          </CardContent>
        </Card>

        {/* Security Config */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Security
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <ConfigRow
              label="Auth Guard"
              value={status?.auth_guard ? "Enabled" : "Disabled"}
            />
            {status?.auth_guard_masked_key && (
              <ConfigRow
                label="API Key"
                value={status.auth_guard_masked_key}
                mono
              />
            )}
          </CardContent>
        </Card>

        {/* Debug Config */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5" />
              Debug
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ConfigRow
              label="Debug Dump"
              value={status?.debug_dump ? "Enabled" : "Disabled"}
            />
          </CardContent>
        </Card>

        {/* Environment Variables */}
        <Card>
          <CardHeader>
            <CardTitle>Environment Variables</CardTitle>
            <CardDescription>
              Configuration can be overridden via environment variables
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-2 text-left font-medium">Variable</th>
                    <th className="px-4 py-2 text-left font-medium">Description</th>
                  </tr>
                </thead>
                <tbody>
                  <EnvRow name="KIRO_CLAUDE_HOST" desc="Server host" />
                  <EnvRow name="KIRO_CLAUDE_PORT" desc="Server port" />
                  <EnvRow name="KIRO_CLAUDE_MODE" desc="Runtime mode" />
                  <EnvRow name="KIRO_CACHE_DIR" desc="Kiro credential cache directory" />
                  <EnvRow name="KIRO_UPSTREAM_ENDPOINT" desc="Upstream endpoint URL" />
                  <EnvRow name="KIRO_PROXY_API_KEY" desc="Proxy API key for auth guard" />
                  <EnvRow name="KIRO_DEBUG_DUMP" desc="Enable debug request/response dump" />
                  <EnvRow name="LOG_LEVEL" desc="Log level (debug, info, warn, error)" />
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function ConfigRow({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className={`text-sm ${mono ? "font-mono" : "font-medium"}`}>
        {value}
      </span>
    </div>
  )
}

function EnvRow({ name, desc }: { name: string; desc: string }) {
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-2 font-mono text-xs">{name}</td>
      <td className="px-4 py-2 text-muted-foreground">{desc}</td>
    </tr>
  )
}
