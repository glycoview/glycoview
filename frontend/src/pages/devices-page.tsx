import { DashboardSection } from "@/components/dashboard/section"
import { MetricStrip } from "@/components/dashboard/metric-strip"
import { DeviceFeedTable, ActivityTable } from "@/components/dashboard/lists"
import { useApiResource } from "@/lib/api"
import type { DevicesResponse } from "@/types"

export function DevicesPage({ token }: { token: string }) {
  const { data, loading, error } = useApiResource<DevicesResponse>("/app/api/devices", token)

  if (loading) return <div className="rounded-2xl border border-border bg-white/88 p-8 text-sm text-slate-500">Loading devices…</div>
  if (error || !data) return <div className="text-sm text-rose-600">{error?.message ?? "Failed to load devices"}</div>

  return (
    <div className="space-y-6">
      <MetricStrip metrics={data.metrics} />
      <DashboardSection title="Device feed overview" subtitle={data.headline}>
        <DeviceFeedTable cards={data.cards} />
      </DashboardSection>
      <DashboardSection title="Recent integration activity">
        <ActivityTable items={data.activity} />
      </DashboardSection>
    </div>
  )
}
