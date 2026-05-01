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
import { HeartPulse, Server, Users, Zap } from "lucide-react"
import type { StatusResponse, EndpointsResponse } from "@/lib/types"

export default function HealthPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null)
  const [endpoints, setEndpoints] = useState<EndpointsResponse | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())

  const refresh = () => {
    Promise.all([
      fetch("/api/status").then((r) => r.json()),
      fetch("/api/endpoints").then((r) => r.json()),
    ])
      .then(([s, e]) => {
        setStatus(s)
        setEndpoints(e)
        setLastRefresh(new Date())
      })
      .catch(() => {})
  }

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, 10000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div>
      <Header
        title="Health Monitor"
        description="Real-time proxy status and endpoint health"
      />
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Auto-refresh every 10 seconds
          </p>
          <p className="text-xs text-muted-foreground">
            Last updated: {lastRefresh.toLocaleTimeString()}
          </p>
        </div>

        {/* Proxy Status */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              Proxy Status
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">Runtime Mode</p>
                <p className="font-medium">{status?.mode || "..."}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">Bind Address</p>
                <p className="font-mono text-sm">{status?.bind_address || "..."}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">Auth Guard</p>
                <Badge variant={status?.auth_guard ? "default" : "secondary"}>
                  {status?.auth_guard ? "Enabled" : "Disabled"}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Account Status */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              Account Status
            </CardTitle>
            <CardDescription>
              Credential health and failover state
            </CardDescription>
          </CardHeader>
          <CardContent>
            {status?.accounts && status.accounts.length > 0 ? (
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-2 text-left font-medium">Account ID</th>
                      <th className="px-4 py-2 text-left font-medium">Auth Type</th>
                      <th className="px-4 py-2 text-left font-medium">Status</th>
                      <th className="px-4 py-2 text-left font-medium">Failures</th>
                      <th className="px-4 py-2 text-left font-medium">Active</th>
                    </tr>
                  </thead>
                  <tbody>
                    {status.accounts.map((acct) => (
                      <tr key={acct.id} className="border-b last:border-0">
                        <td className="px-4 py-2 font-mono text-xs">{acct.id}</td>
                        <td className="px-4 py-2">{acct.auth_type}</td>
                        <td className="px-4 py-2">
                          <Badge variant={acct.healthy ? "default" : "destructive"}>
                            {acct.healthy ? "Healthy" : "Unhealthy"}
                          </Badge>
                        </td>
                        <td className="px-4 py-2">{acct.failures}</td>
                        <td className="px-4 py-2">
                          {acct.is_current && (
                            <Badge variant="outline">Current</Badge>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No accounts loaded</p>
            )}
          </CardContent>
        </Card>

        {/* Endpoint Health */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="h-5 w-5" />
              Endpoint Health
            </CardTitle>
            <CardDescription>
              Self-check results for each API endpoint
            </CardDescription>
          </CardHeader>
          <CardContent>
            {endpoints?.endpoints ? (
              <div className="space-y-2">
                {endpoints.endpoints.map((ep) => (
                  <div
                    key={ep.path}
                    className="flex items-center justify-between rounded-md border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <Badge variant="outline">{ep.method}</Badge>
                      <code className="text-sm font-mono">{ep.path}</code>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-muted-foreground">
                        {ep.latency_ms}ms
                      </span>
                      <Badge
                        variant={
                          ep.status === "ok"
                            ? "default"
                            : ep.status === "degraded"
                            ? "secondary"
                            : "destructive"
                        }
                      >
                        {ep.status}
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Loading endpoints...</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
