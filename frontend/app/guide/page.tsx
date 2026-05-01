"use client"

import { Header } from "@/components/layout/header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Terminal, Globe, Puzzle, ArrowRight } from "lucide-react"
import Link from "next/link"

const clients = [
  {
    name: "Claude Code",
    href: "/guide/claude-code",
    icon: Terminal,
    description: "Anthropic's official CLI tool for Claude",
    difficulty: "Easy",
  },
  {
    name: "Cursor",
    href: "/guide/cursor",
    icon: Puzzle,
    description: "AI-powered code editor",
    difficulty: "Easy",
  },
  {
    name: "Cline",
    href: "/guide/claude-code",
    icon: Globe,
    description: "VS Code extension for AI coding",
    difficulty: "Easy",
  },
]

export default function GuidePage() {
  return (
    <div>
      <Header
        title="Integration Guide"
        description="Step-by-step instructions to connect your AI coding tools"
      />
      <div className="p-6 space-y-6">
        {/* Prerequisites */}
        <Card>
          <CardHeader>
            <CardTitle>Prerequisites</CardTitle>
            <CardDescription>
              What you need before getting started
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <Badge variant="outline" className="mt-0.5">1</Badge>
              <div>
                <p className="font-medium">Kiro Claude Proxy running</p>
                <p className="text-sm text-muted-foreground">
                  The Go backend must be running on port 8000 (or your configured port).
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <Badge variant="outline" className="mt-0.5">2</Badge>
              <div>
                <p className="font-medium">Kiro credentials configured</p>
                <p className="text-sm text-muted-foreground">
                  Either Kiro Desktop credentials in ~/.aws/sso/cache, or KIRO_REFRESH_TOKEN environment variable.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <Badge variant="outline" className="mt-0.5">3</Badge>
              <div>
                <p className="font-medium">Client tool installed</p>
                <p className="text-sm text-muted-foreground">
                  Claude Code CLI, Cursor, Cline, or any Anthropic-compatible client.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Client Selection */}
        <div className="grid gap-4 md:grid-cols-3">
          {clients.map((client) => (
            <Link key={client.name} href={client.href}>
              <Card className="h-full transition-colors hover:border-primary hover:bg-primary/5 cursor-pointer">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <client.icon className="h-8 w-8 text-primary" />
                    <Badge variant="secondary">{client.difficulty}</Badge>
                  </div>
                  <CardTitle className="mt-2">{client.name}</CardTitle>
                  <CardDescription>{client.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center text-sm text-primary font-medium">
                    Setup guide <ArrowRight className="ml-1 h-4 w-4" />
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        {/* Credential Setup */}
        <Card>
          <CardHeader>
            <CardTitle>Credential Setup</CardTitle>
            <CardDescription>
              How to configure Kiro authentication
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="font-medium mb-2">Kiro Desktop (Recommended)</h4>
              <p className="text-sm text-muted-foreground mb-2">
                If you have Kiro IDE installed, credentials are automatically stored in:
              </p>
              <code className="block rounded bg-muted px-4 py-2 text-sm font-mono">
                ~/.aws/sso/cache/kiro-auth-token.json
              </code>
              <p className="mt-2 text-sm text-muted-foreground">
                The proxy auto-detects these credentials. No additional configuration needed.
              </p>
            </div>
            <div>
              <h4 className="font-medium mb-2">AWS SSO (OIDC)</h4>
              <p className="text-sm text-muted-foreground mb-2">
                For AWS SSO / Builder ID authentication, your credentials JSON must include clientId and clientSecret:
              </p>
              <code className="block rounded bg-muted px-4 py-2 text-sm font-mono whitespace-pre">{`{
  "clientId": "...",
  "clientSecret": "...",
  "refreshToken": "...",
  "region": "us-east-1"
}`}</code>
            </div>
            <div>
              <h4 className="font-medium mb-2">Environment Variable</h4>
              <p className="text-sm text-muted-foreground mb-2">
                Set KIRO_REFRESH_TOKEN as a fallback:
              </p>
              <code className="block rounded bg-muted px-4 py-2 text-sm font-mono">
                export KIRO_REFRESH_TOKEN=your-token-here
              </code>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
