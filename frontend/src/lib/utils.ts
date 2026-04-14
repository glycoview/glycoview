import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}

export function formatMinutes(minutes: number) {
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return "0m"
  }

  const hours = Math.floor(minutes / 60)
  const remainder = Math.round(minutes % 60)

  if (!hours) {
    return `${remainder}m`
  }

  if (!remainder) {
    return `${hours}h`
  }

  return `${hours}h ${remainder}m`
}

export function formatRelativeTime(value: number) {
  const delta = Math.max(0, Date.now() - value)
  const minutes = Math.floor(delta / 60000)

  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`

  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`

  return new Date(value).toLocaleDateString([], { month: "short", day: "2-digit" })
}

export function formatValue(value: number, unit: string) {
  if (!Number.isFinite(value)) {
    return `0 ${unit}`.trim()
  }

  const rounded = Math.abs(value) >= 10 ? Math.round(value) : Number(value.toFixed(1))
  return `${rounded} ${unit}`.trim()
}
