import { useState } from "react"

import { login } from "@/lib/api"
import type { ApiError } from "@/lib/api"
import { AuthShell } from "@/components/layout/auth-shell"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function LoginPage({ onSuccess }: { onSuccess: () => Promise<unknown> }) {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string>("")
  const [submitting, setSubmitting] = useState(false)

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    setError("")
    try {
      await login(username, password)
      await onSuccess()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Login failed")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthShell title="Login" subtitle="Sign in to the dashboard with your account credentials.">
      <div className="text-2xl font-semibold tracking-tight text-slate-950">Sign in</div>
      <div className="mt-2 text-sm text-slate-500">Username and password are required.</div>
      <form onSubmit={submit} className="mt-6 space-y-4">
        <Input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="Username" />
        <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Password" />
        {error ? <div className="text-sm text-rose-600">{error}</div> : null}
        <Button className="w-full" disabled={submitting}>
          {submitting ? "Signing in..." : "Sign in"}
        </Button>
      </form>
    </AuthShell>
  )
}
