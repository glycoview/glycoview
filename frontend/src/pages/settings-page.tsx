import { type ReactNode, useEffect, useMemo, useState } from "react"

import {
  applyUpdate,
  configureTLS,
  fetchSettingsStatus,
  fetchTLSConfig,
  fetchTLSProviders,
  fetchUpdateCheck,
  rollbackUpdate,
} from "@/lib/api"
import type { ApiError } from "@/lib/api"
import type { ApplianceStatus, ApplianceTLSConfig, TLSProvider, UpdateCheckResponse } from "@/types"
import { DashboardSection } from "@/components/dashboard/section"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

type TLSForm = {
  domain: string
  email: string
  challengeType: "http-01" | "dns-01"
  provider: string
  env: Record<string, string>
}

export function SettingsPage() {
  const [status, setStatus] = useState<ApplianceStatus | null>(null)
  const [update, setUpdate] = useState<UpdateCheckResponse | null>(null)
  const [providers, setProviders] = useState<TLSProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState("")
  const [error, setError] = useState("")
  const [message, setMessage] = useState("")
  const [updateTag, setUpdateTag] = useState("")
  const [includeAgent, setIncludeAgent] = useState(true)
  const [tlsForm, setTLSForm] = useState<TLSForm>({
    domain: "",
    email: "",
    challengeType: "http-01",
    provider: "",
    env: {},
  })

  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === tlsForm.provider) ?? null,
    [providers, tlsForm.provider],
  )

  const load = async () => {
    setLoading(true)
    try {
      const [statusResponse, updateResponse, providersResponse, tlsResponse] = await Promise.all([
        fetchSettingsStatus(),
        fetchUpdateCheck(),
        fetchTLSProviders(),
        fetchTLSConfig(),
      ])
      setStatus(statusResponse)
      setUpdate(updateResponse)
      setProviders(providersResponse.providers)
      setUpdateTag(updateResponse.latestTag || statusResponse.currentTag)
      setTLSForm({
        domain: tlsResponse.domain || "",
        email: tlsResponse.email || "",
        challengeType: tlsResponse.challengeType === "dns-01" ? "dns-01" : "http-01",
        provider: tlsResponse.provider || providersResponse.providers[0]?.id || "",
        env: tlsResponse.env || {},
      })
      setError("")
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to load settings")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load().catch(() => undefined)
  }, [])

  const submitTLS = async (event: React.FormEvent) => {
    event.preventDefault()
    setBusy("tls")
    setMessage("")
    setError("")
    try {
      const payload: ApplianceTLSConfig = {
        domain: tlsForm.domain,
        email: tlsForm.email,
        challengeType: tlsForm.challengeType,
        provider: tlsForm.challengeType === "dns-01" ? tlsForm.provider : "",
        env: tlsForm.challengeType === "dns-01" ? tlsForm.env : {},
      }
      const result = await configureTLS(payload)
      setMessage(result.message)
      await load()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to save TLS settings")
    } finally {
      setBusy("")
    }
  }

  const submitUpdate = async () => {
    setBusy("update")
    setMessage("")
    setError("")
    try {
      const result = await applyUpdate({ tag: updateTag, includeAgent })
      setMessage(result.message)
      await load()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to apply update")
    } finally {
      setBusy("")
    }
  }

  const submitRollback = async () => {
    setBusy("rollback")
    setMessage("")
    setError("")
    try {
      const result = await rollbackUpdate()
      setMessage(result.message)
      await load()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to roll back update")
    } finally {
      setBusy("")
    }
  }

  return (
    <div className="space-y-6">
      {error ? <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</div> : null}
      {message ? <div className="rounded-lg border border-border bg-card px-4 py-3 text-sm text-foreground">{message}</div> : null}

      <DashboardSection title="System">
        {loading || !status ? (
          <div className="text-sm text-muted-foreground">Loading…</div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <Metric label="Current app tag" value={status.currentTag} />
            <Metric label="Current agent tag" value={status.currentAgentTag || "n/a"} />
            <Metric label="Docker control" value={status.dockerManaged ? "Available" : "Unavailable"} />
            <Metric label="Last action" value={status.lastAction || "No actions yet"} detail={status.lastMessage} />
          </div>
        )}
      </DashboardSection>

      <DashboardSection title="Updates">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">Target tag</label>
              <Input value={updateTag} onChange={(event) => setUpdateTag(event.target.value)} placeholder="latest" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">Latest release</label>
              <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm text-foreground">
                {update?.latestTag || "Unavailable"}
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm text-foreground md:col-span-2">
              <input
                type="checkbox"
                checked={includeAgent}
                onChange={(event) => setIncludeAgent(event.target.checked)}
                className="h-4 w-4 rounded border-border text-primary"
              />
              Update the appliance agent to the same tag
            </label>
            {update?.releaseUrl ? (
              <a className="text-sm font-medium text-primary hover:underline md:col-span-2" href={update.releaseUrl} target="_blank" rel="noreferrer">
                Open release notes
              </a>
            ) : null}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={() => load().catch(() => undefined)} disabled={busy !== ""}>
              Refresh
            </Button>
            <Button type="button" variant="outline" onClick={submitRollback} disabled={busy !== ""}>
              {busy === "rollback" ? "Rolling back…" : "Rollback"}
            </Button>
            <Button type="button" onClick={submitUpdate} disabled={busy !== "" || !updateTag.trim()}>
              {busy === "update" ? "Applying…" : "Apply update"}
            </Button>
          </div>
        </div>
      </DashboardSection>

      <DashboardSection title="TLS and domain">
        <form onSubmit={submitTLS} className="space-y-5">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <Field label="Domain">
              <Input value={tlsForm.domain} onChange={(event) => setTLSForm((prev) => ({ ...prev, domain: event.target.value }))} placeholder="glycoview.example.com" />
            </Field>
            <Field label="Let's Encrypt email">
              <Input value={tlsForm.email} onChange={(event) => setTLSForm((prev) => ({ ...prev, email: event.target.value }))} placeholder="admin@example.com" />
            </Field>
            <Field label="Challenge">
              <select
                value={tlsForm.challengeType}
                onChange={(event) =>
                  setTLSForm((prev) => ({
                    ...prev,
                    challengeType: event.target.value === "dns-01" ? "dns-01" : "http-01",
                    provider: event.target.value === "dns-01" ? prev.provider || providers[0]?.id || "" : "",
                  }))
                }
                className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
              >
                <option value="http-01">HTTP challenge</option>
                <option value="dns-01">DNS challenge</option>
              </select>
            </Field>
            <Field label="DNS provider">
              <select
                value={tlsForm.provider}
                onChange={(event) => setTLSForm((prev) => ({ ...prev, provider: event.target.value }))}
                disabled={tlsForm.challengeType !== "dns-01"}
                className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-60"
              >
                <option value="">Select provider</option>
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.id}>
                    {provider.label}
                  </option>
                ))}
              </select>
            </Field>
          </div>

          {tlsForm.challengeType === "dns-01" && selectedProvider ? (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {selectedProvider.fields.map((field) => (
                <Field key={field.key} label={field.label}>
                  <Input
                    type={field.secret ? "password" : "text"}
                    value={tlsForm.env[field.key] || ""}
                    placeholder={field.placeholder}
                    onChange={(event) =>
                      setTLSForm((prev) => ({
                        ...prev,
                        env: {
                          ...prev.env,
                          [field.key]: event.target.value,
                        },
                      }))
                    }
                  />
                </Field>
              ))}
            </div>
          ) : null}

          <div className="flex items-center justify-between gap-3 border-t border-border pt-4">
            <div className="text-sm text-muted-foreground">
              {status?.tls.appliedAt ? `Last applied ${new Date(status.tls.appliedAt).toLocaleString()}` : "No TLS configuration has been applied yet."}
            </div>
            <Button type="submit" disabled={busy !== ""}>
              {busy === "tls" ? "Saving…" : "Save TLS settings"}
            </Button>
          </div>
        </form>
      </DashboardSection>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <label className="text-sm font-medium text-foreground">{label}</label>
      {children}
    </div>
  )
}

function Metric({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div className="space-y-1 rounded-lg border border-border bg-muted/30 px-4 py-3">
      <div className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{label}</div>
      <div className="text-sm font-semibold text-foreground">{value}</div>
      {detail ? <div className="text-sm text-muted-foreground">{detail}</div> : null}
    </div>
  )
}
