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
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search, ArrowRight, Cpu } from "lucide-react"
import type { ModelsResponse, ResolveResponse } from "@/lib/types"

export default function ModelsPage() {
  const [models, setModels] = useState<ModelsResponse | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [resolveResult, setResolveResult] = useState<ResolveResponse | null>(null)
  const [resolveLoading, setResolveLoading] = useState(false)
  const [resolveError, setResolveError] = useState<string | null>(null)

  useEffect(() => {
    fetch("/api/models")
      .then((res) => res.json())
      .then(setModels)
      .catch(() => {})
  }, [])

  const handleResolve = () => {
    if (!searchQuery.trim()) return
    setResolveLoading(true)
    setResolveError(null)
    setResolveResult(null)

    fetch("/api/resolve-model", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model: searchQuery.trim() }),
    })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json()
      })
      .then(setResolveResult)
      .catch((err) => setResolveError(err.message))
      .finally(() => setResolveLoading(false))
  }

  return (
    <div>
      <Header
        title="Model Detection"
        description="View supported models and test name resolution"
      />
      <div className="p-6 space-y-6">
        {/* Model Resolver Tester */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5" />
              Model Name Resolver
            </CardTitle>
            <CardDescription>
              Enter any Claude model name to see how the proxy resolves it
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input
                placeholder="e.g. claude-sonnet-4-5-20250929"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleResolve()}
                className="font-mono text-sm"
              />
              <Button onClick={handleResolve} disabled={resolveLoading}>
                {resolveLoading ? "Resolving..." : "Resolve"}
              </Button>
            </div>

            {resolveError && (
              <p className="text-sm text-destructive">{resolveError}</p>
            )}

            {resolveResult && (
              <div className="space-y-4">
                {/* Pipeline Visualization */}
                <div className="rounded-lg border p-4">
                  <h4 className="text-sm font-medium mb-3">Resolution Pipeline</h4>
                  <div className="flex items-center gap-2 flex-wrap">
                    {resolveResult.pipeline.map((step, i) => (
                      <div key={i} className="flex items-center gap-2">
                        <div className="rounded-md border bg-card p-2 text-center">
                          <div className="text-xs text-muted-foreground mb-1">
                            {step.step}
                          </div>
                          <code className="text-xs font-mono">{step.output}</code>
                        </div>
                        {i < resolveResult.pipeline.length - 1 && (
                          <ArrowRight className="h-4 w-4 text-muted-foreground shrink-0" />
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                {/* Result Details */}
                <div className="grid gap-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Original</span>
                    <code className="font-mono">{resolveResult.original}</code>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Normalized</span>
                    <code className="font-mono">{resolveResult.normalized}</code>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Internal ID</span>
                    <code className="font-mono text-xs">{resolveResult.internal_id}</code>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-muted-foreground">Source</span>
                    <Badge variant={resolveResult.source === "static" ? "default" : "secondary"}>
                      {resolveResult.source}
                    </Badge>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Model List */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cpu className="h-5 w-5" />
              Available Models
            </CardTitle>
            <CardDescription>
              Models reported by the proxy via GET /v1/models
            </CardDescription>
          </CardHeader>
          <CardContent>
            {models ? (
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-2 text-left font-medium">Model ID</th>
                      <th className="px-4 py-2 text-left font-medium">Provider</th>
                    </tr>
                  </thead>
                  <tbody>
                    {models.data.map((model) => (
                      <tr key={model.id} className="border-b last:border-0">
                        <td className="px-4 py-2 font-mono text-xs">{model.id}</td>
                        <td className="px-4 py-2 text-muted-foreground">
                          {model.owned_by}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Loading models...</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
