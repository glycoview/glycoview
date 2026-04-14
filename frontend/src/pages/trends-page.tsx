import { useMemo, useState } from "react"

import { DashboardSection } from "@/components/dashboard/section"
import { MetricStrip } from "@/components/dashboard/metric-strip"
import { AgpChart } from "@/components/dashboard/charts"
import { TimeInRange } from "@/components/dashboard/time-in-range"
import { DailySummaryTable } from "@/components/dashboard/lists"
import { Button } from "@/components/ui/button"
import { useApiResource } from "@/lib/api"
import type { TrendsResponse } from "@/types"

const windows = [7, 14, 30]

export function TrendsPage({ token }: { token: string }) {
  const [days, setDays] = useState(14)
  const path = useMemo(() => `/app/api/trends?days=${days}`, [days])
  const { data, loading, error } = useApiResource<TrendsResponse>(path, token)

  if (loading) return <div className="rounded-2xl border border-border bg-white/88 p-8 text-sm text-slate-500">Loading trend analysis...</div>
  if (error || !data) return <div className="text-sm text-rose-600">{error?.message ?? "Failed to load trends"}</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/80 bg-white/70 px-4 py-3">
        <div className="text-sm font-medium text-slate-600">Window</div>
        <div className="inline-flex flex-wrap gap-2 rounded-xl bg-slate-100/80 p-1">
          {windows.map((windowDays) => (
            <Button
              key={windowDays}
              size="sm"
              variant={windowDays === days ? "default" : "ghost"}
              className={windowDays === days ? "shadow-none" : "bg-transparent"}
              onClick={() => setDays(windowDays)}
            >
              {windowDays} days
            </Button>
          ))}
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.85fr)]">
        <DashboardSection title="AGP profile" subtitle={data.rangeLabel}>
          <AgpChart buckets={data.agp} />
        </DashboardSection>
        <DashboardSection title="Range distribution">
          <TimeInRange bands={data.timeInRange} />
        </DashboardSection>
      </div>

      <MetricStrip metrics={data.metrics} />

      <DashboardSection title="Daily summary">
        <DailySummaryTable rows={data.daysSummary} />
      </DashboardSection>
    </div>
  )
}
