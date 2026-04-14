import { useEffect, useMemo, useState } from "react"
import { Navigate, Route, Routes } from "react-router-dom"

import { AppShell } from "@/components/layout/app-shell"
import { fetchAuthStatus, logout as logoutRequest, readStoredToken } from "@/lib/api"
import type { AuthStatus } from "@/types"
import { OverviewPage } from "@/pages/overview-page"
import { DailyPage } from "@/pages/daily-page"
import { TrendsPage } from "@/pages/trends-page"
import { ProfilePage } from "@/pages/profile-page"
import { DevicesPage } from "@/pages/devices-page"
import { LoginPage } from "@/pages/login-page"
import { SetupPage } from "@/pages/setup-page"
import { UsersPage } from "@/pages/users-page"
import { SettingsPage } from "@/pages/settings-page"

function App() {
  const [token] = useState(() => readStoredToken())
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null)
  const [authLoading, setAuthLoading] = useState(true)

  const refreshAuth = useMemo(
    () => () =>
      fetchAuthStatus()
        .then((status) => {
          setAuthStatus(status)
          setAuthLoading(false)
          return status
        }),
    [],
  )

  useEffect(() => {
    refreshAuth().catch(() => {
      setAuthStatus({ authenticated: false, setupRequired: false })
      setAuthLoading(false)
    })
  }, [refreshAuth])

  if (authLoading || !authStatus) {
    return <div className="flex min-h-screen items-center justify-center bg-background text-sm text-slate-500">Loading…</div>
  }

  if (authStatus.setupRequired) {
    return <SetupPage onComplete={refreshAuth} />
  }

  if (!authStatus.authenticated || !authStatus.user) {
    return <LoginPage onSuccess={refreshAuth} />
  }

  const shellProps = {
    user: authStatus.user,
    onLogout: async () => {
      await logoutRequest()
      await refreshAuth()
    },
  }

  return (
    <Routes>
      <Route element={<AppShell {...shellProps} />}>
        <Route path="/" element={<OverviewPage token={token} />} />
        <Route path="/daily" element={<DailyPage token={token} />} />
        <Route path="/trends" element={<TrendsPage token={token} />} />
        <Route path="/profile" element={<ProfilePage token={token} />} />
        <Route path="/devices" element={<DevicesPage token={token} />} />
        {authStatus.user.role === "admin" ? <Route path="/users" element={<UsersPage />} /> : null}
        {authStatus.user.role === "admin" ? <Route path="/settings" element={<SettingsPage />} /> : null}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default App
