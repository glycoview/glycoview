import { useState } from "react"

import { setup } from "@/lib/api"
import type { ApiError } from "@/lib/api"
import { AuthShell } from "@/components/layout/auth-shell"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function SetupPage({ onComplete }: { onComplete: () => Promise<unknown> }) {
  const [displayName, setDisplayName] = useState("")
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [apiSecret, setApiSecret] = useState("")
  const [error, setError] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    setError("")
    try {
      const response = await setup(username, password, displayName)
      setApiSecret(response.apiSecret)
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Setup failed")
    } finally {
      setSubmitting(false)
    }
  }

  if (apiSecret) {
    return (
      <AuthShell title="Setup complete" subtitle="The dashboard account is ready and the install API secret has been generated.">
        <div className="text-2xl font-semibold tracking-tight text-slate-950">Save the Nightscout API secret</div>
        <div className="mt-2 text-sm text-slate-500">
          Use this secret for local development or compatible Nightscout clients. This value is tied to this install.
        </div>
        <div className="mt-6 rounded-xl border border-border bg-slate-50 px-4 py-4">
          <div className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">API secret</div>
          <code className="mt-3 block overflow-x-auto text-sm font-medium text-slate-950">{apiSecret}</code>
        </div>
        <div className="mt-4 flex gap-3">
          <Button type="button" variant="outline" onClick={() => navigator.clipboard?.writeText(apiSecret)}>
            Copy
          </Button>
          <Button type="button" onClick={() => void onComplete()}>
            Continue to dashboard
          </Button>
        </div>
      </AuthShell>
    )
  }

  return (
    <AuthShell title="Create first admin" subtitle="Set up the first dashboard account for this self-hosted installation.">
      <div className="text-2xl font-semibold tracking-tight text-slate-950">Create first admin</div>
      <div className="mt-2 text-sm text-slate-500">This step is shown only on a fresh install with no existing users.</div>
      <form onSubmit={submit} className="mt-6 space-y-4">
        <Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Display name" />
        <Input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="Username" />
        <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Password" />
        {error ? <div className="text-sm text-rose-600">{error}</div> : null}
        <Button className="w-full" disabled={submitting}>
          {submitting ? "Creating..." : "Create admin"}
        </Button>
      </form>
    </AuthShell>
  )
}
