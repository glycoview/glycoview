import { useEffect, useState } from "react"

import type {
  AppUser,
  ApplianceActionResponse,
  ApplianceStatus,
  ApplianceTLSConfig,
  AuthStatus,
  TLSProvider,
  UpdateCheckResponse,
} from "@/types"

const TOKEN_KEY = "glycoview.api-token"

export type ApiError = {
  status: number
  message: string
}

export function readStoredToken() {
  return window.localStorage.getItem(TOKEN_KEY) ?? ""
}

export function writeStoredToken(token: string) {
  window.localStorage.setItem(TOKEN_KEY, token.trim())
}

function headersForToken(token: string) {
  const headers = new Headers()
  const trimmed = token.trim()
  if (!trimmed) return headers
  if (trimmed.split(".").length === 3) {
    headers.set("Authorization", `Bearer ${trimmed}`)
  } else {
    headers.set("api-secret", trimmed)
  }
  return headers
}

export async function fetchJson<T>(path: string, token: string): Promise<T> {
  const response = await fetch(path, {
    headers: headersForToken(token),
    credentials: "include",
  })

  if (!response.ok) {
    let message = response.statusText
    try {
      const payload = await response.json()
      if (typeof payload.message === "string") {
        message = payload.message
      }
    } catch {
      // ignore
    }
    throw { status: response.status, message } satisfies ApiError
  }

  return response.json() as Promise<T>
}

export async function postJson<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  })
  return handleJsonResponse<T>(response)
}

export async function patchJson<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "PATCH",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  })
  return handleJsonResponse<T>(response)
}

async function handleJsonResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = response.statusText
    try {
      const payload = await response.json()
      if (typeof payload.message === "string") {
        message = payload.message
      }
    } catch {
      // ignore
    }
    throw { status: response.status, message } satisfies ApiError
  }
  return response.json() as Promise<T>
}

export function fetchAuthStatus() {
  return fetchJson<AuthStatus>("/app/api/auth/status", "")
}

export function login(username: string, password: string) {
  return postJson<{ user: AppUser }>("/app/api/auth/login", { username, password })
}

export function setup(username: string, password: string, displayName: string) {
  return postJson<{ user: AppUser; apiSecret: string }>("/app/api/auth/setup", { username, password, displayName })
}

export function logout() {
  return postJson<{ status: string }>("/app/api/auth/logout", {})
}

export function fetchInstallSecret() {
  return fetchJson<{ apiSecret: string }>("/app/api/auth/install-secret", "")
}

export function fetchUsers() {
  return fetchJson<{ users: AppUser[] }>("/app/api/users", "")
}

export function createUser(input: { username: string; password: string; displayName: string; role: string }) {
  return postJson<{ user: AppUser }>("/app/api/users", input)
}

export function updateUser(id: string, input: { displayName?: string; password?: string; role?: string; active?: boolean }) {
  return patchJson<{ user: AppUser }>(`/app/api/users/${id}`, input)
}

export function fetchSettingsStatus() {
  return fetchJson<ApplianceStatus>("/app/api/settings/status", "")
}

export function fetchUpdateCheck() {
  return fetchJson<UpdateCheckResponse>("/app/api/settings/updates/check", "")
}

export function applyUpdate(input: { tag: string; includeAgent?: boolean }) {
  return postJson<ApplianceActionResponse>("/app/api/settings/updates/apply", input)
}

export function rollbackUpdate() {
  return postJson<ApplianceActionResponse>("/app/api/settings/updates/rollback", {})
}

export function fetchTLSProviders() {
  return fetchJson<{ providers: TLSProvider[] }>("/app/api/settings/tls/providers", "")
}

export function fetchTLSConfig() {
  return fetchJson<ApplianceTLSConfig>("/app/api/settings/tls/config", "")
}

export function configureTLS(input: ApplianceTLSConfig) {
  return postJson<ApplianceActionResponse>("/app/api/settings/tls/configure", input)
}

export function useApiResource<T>(path: string, token: string) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<ApiError | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    fetchJson<T>(path, token)
      .then((next) => {
        if (!cancelled) {
          setData(next)
          setLoading(false)
        }
      })
      .catch((err: ApiError) => {
        if (!cancelled) {
          setError(err)
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [path, token])

  return { data, loading, error }
}
