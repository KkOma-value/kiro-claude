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
import { Copy, Check, Puzzle } from "lucide-react"

export default function CursorGuidePage() {
  const [copiedStep, setCopiedStep] = useState<number | null>(null)

  const copy = (text: string, step: number) => {
    navigator.clipboard.writeText(text)
    setCopiedStep(step)
    setTimeout(() => setCopiedStep(null), 2000)
  }

  return (
    <div>
      <Header
        title="Cursor Setup"
        description="Connect Cursor IDE to the Kiro proxy"
      />
      <div className="p-6 space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Puzzle className="h-8 w-8 text-primary" />
              <div>
                <CardTitle>Cursor</CardTitle>
                <CardDescription>AI-powered code editor</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Cursor supports custom API endpoints for Claude models. You can point it at
              the Kiro proxy to use Claude through Kiro&apos;s infrastructure.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Badge variant="outline">1</Badge>
              <CardTitle className="text-base">Open Cursor Settings</CardTitle>
            </div>
            <CardDescription>
              Navigate to Settings &gt; Models in Cursor
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Open Cursor, go to Settings (Ctrl+,), then navigate to the &quot;Models&quot; section.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Badge variant="outline">2</Badge>
              <CardTitle className="text-base">Set API Base URL</CardTitle>
            </div>
            <CardDescription>
              Configure the custom API endpoint
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded bg-muted px-4 py-2 text-sm font-mono">
                http://127.0.0.1:8000
              </code>
              <button
                onClick={() => copy("http://127.0.0.1:8000", 2)}
                className="rounded-md p-2 hover:bg-accent transition-colors"
              >
                {copiedStep === 2 ? (
                  <Check className="h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground" />
                )}
              </button>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Set this as the API Base URL for Anthropic/Claude models.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Badge variant="outline">3</Badge>
              <CardTitle className="text-base">Set API Key</CardTitle>
            </div>
            <CardDescription>
              If the proxy has auth guard enabled, enter the API key
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Enter your proxy API key in the API Key field. If auth guard is disabled,
              you can leave this empty or enter any value.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
