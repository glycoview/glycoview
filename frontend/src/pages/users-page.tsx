import { useEffect, useState } from "react"

import { createUser, fetchInstallSecret, fetchUsers, updateUser } from "@/lib/api"
import type { ApiError } from "@/lib/api"
import type { AppUser } from "@/types"
import { DashboardSection } from "@/components/dashboard/section"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function UsersPage() {
  const [users, setUsers] = useState<AppUser[]>([])
  const [apiSecret, setApiSecret] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState({ displayName: "", username: "", password: "", role: "doctor" })

  const load = async () => {
    setLoading(true)
    try {
      const [usersResponse, secretResponse] = await Promise.all([fetchUsers(), fetchInstallSecret()])
      setUsers(usersResponse.users)
      setApiSecret(secretResponse.apiSecret)
      setError("")
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to load users")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load().catch(() => undefined)
  }, [])

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    try {
      await createUser(form)
      setForm({ displayName: "", username: "", password: "", role: "doctor" })
      await load()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to create user")
    }
  }

  const toggleActive = async (user: AppUser) => {
    try {
      await updateUser(user.id, { active: !user.active })
      await load()
    } catch (err) {
      const apiError = err as ApiError
      setError(apiError.message || "Failed to update user")
    }
  }

  return (
    <div className="space-y-6">
      <DashboardSection title="API access">
        <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto]">
          <Input readOnly value={apiSecret} />
          <Button type="button" variant="outline" onClick={() => navigator.clipboard?.writeText(apiSecret)} disabled={!apiSecret}>
            Copy
          </Button>
        </div>
      </DashboardSection>

      <DashboardSection title="Create user">
        <form onSubmit={submit} className="grid gap-3 xl:grid-cols-5">
          <Input value={form.displayName} onChange={(event) => setForm((prev) => ({ ...prev, displayName: event.target.value }))} placeholder="Display name" />
          <Input value={form.username} onChange={(event) => setForm((prev) => ({ ...prev, username: event.target.value }))} placeholder="Username" />
          <Input type="password" value={form.password} onChange={(event) => setForm((prev) => ({ ...prev, password: event.target.value }))} placeholder="Password" />
          <select
            value={form.role}
            onChange={(event) => setForm((prev) => ({ ...prev, role: event.target.value }))}
            className="flex h-10 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm text-slate-900"
          >
            <option value="doctor">Doctor</option>
            <option value="admin">Admin</option>
          </select>
          <Button type="submit">Create</Button>
        </form>
      </DashboardSection>

      <DashboardSection title="Users">
        {loading ? <div className="text-sm text-slate-500">Loading…</div> : null}
        {error ? <div className="mb-4 text-sm text-rose-600">{error}</div> : null}
        <div className="divide-y divide-border">
          {users.map((user) => (
            <div key={user.id} className="flex flex-col gap-3 py-4 md:flex-row md:items-center md:justify-between">
              <div>
                <div className="text-sm font-semibold text-slate-950">{user.displayName}</div>
                <div className="text-sm text-slate-500">{user.username} · {user.role}</div>
              </div>
              <div className="flex items-center gap-2">
                <div className="text-sm text-slate-500">{user.active ? "Active" : "Disabled"}</div>
                <Button size="sm" variant="outline" onClick={() => toggleActive(user)}>
                  {user.active ? "Disable" : "Enable"}
                </Button>
              </div>
            </div>
          ))}
        </div>
      </DashboardSection>
    </div>
  )
}
