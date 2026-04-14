import { useMemo, useState } from "react"

import { DashboardSection } from "@/components/dashboard/section"
import { MetricStrip } from "@/components/dashboard/metric-strip"
import { GlucoseSparkline } from "@/components/dashboard/charts"
import { TimeInRange } from "@/components/dashboard/time-in-range"
import { DeviceFeedTable, EventTable } from "@/components/dashboard/lists"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useApiResource } from "@/lib/api"
import type { DailyResponse } from "@/types"

export function DailyPage({ token }: { token: string }) {
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10))
  const path = useMemo(() => `/app/api/daily?date=${date}`, [date])
  const { data, loading, error } = useApiResource<DailyResponse>(path, token)

  const shift = (days: number) => {
    const base = new Date(`${date}T00:00:00Z`)
    base.setUTCDate(base.getUTCDate() + days)
    setDate(base.toISOString().slice(0, 10))
  }

  if (loading) return <Skeleton className="h-[600px] rounded-2xl" />
  if (error || !data) return <div className="text-sm text-rose-600">{error?.message ?? "Failed to load day"}</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/80 bg-white/70 px-4 py-3">
        <div className="text-sm font-medium text-slate-600">{date}</div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" onClick={() => shift(-1)}>Previous</Button>
          <Button size="sm" variant="outline" onClick={() => shift(1)}>Next</Button>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.85fr)]">
        <DashboardSection title="Daily glucose trace" subtitle={data.dateLabel}>
          <GlucoseSparkline points={data.glucose} tall />
        </DashboardSection>
        <DashboardSection title="Time in range">
          <TimeInRange bands={data.timeInRange} />
        </DashboardSection>
      </div>

      <MetricStrip metrics={data.metrics} />

      <div className="grid gap-6 xl:grid-cols-2">
        <DashboardSection title="Carbohydrates">
          <EventTable events={data.carbs} unit="g" emptyText="No carbohydrate events recorded for this day." />
        </DashboardSection>
        <DashboardSection title="Insulin">
          <EventTable events={data.insulin} unit="U" emptyText="No insulin events recorded for this day." />
        </DashboardSection>
      </div>

      <DashboardSection title="Devices">
        <DeviceFeedTable cards={data.devices} />
      </DashboardSection>
    </div>
  )
}
