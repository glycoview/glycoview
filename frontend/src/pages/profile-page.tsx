import { DashboardSection } from "@/components/dashboard/section"
import { MetricStrip } from "@/components/dashboard/metric-strip"
import { ScheduleTable, ActivityTable } from "@/components/dashboard/lists"
import { useApiResource } from "@/lib/api"
import type { ProfileResponse } from "@/types"

export function ProfilePage({ token }: { token: string }) {
  const { data, loading, error } = useApiResource<ProfileResponse>("/app/api/profile", token)

  if (loading) return <div className="rounded-2xl border border-border bg-white/88 p-8 text-sm text-slate-500">Loading profile…</div>
  if (error || !data) return <div className="text-sm text-rose-600">{error?.message ?? "Failed to load profile"}</div>

  return (
    <div className="space-y-6">
      <MetricStrip metrics={data.metrics} />

      <div className="grid gap-6 xl:grid-cols-2">
        <DashboardSection title="Basal schedule" subtitle={data.headline}>
          <ScheduleTable rows={data.basalSchedule} />
        </DashboardSection>
        <DashboardSection title="Carb ratios">
          <ScheduleTable rows={data.carbRatios} />
        </DashboardSection>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <DashboardSection title="Sensitivity">
          <ScheduleTable rows={data.sensitivity} />
        </DashboardSection>
        <DashboardSection title="Targets">
          <ScheduleTable rows={data.targets} />
        </DashboardSection>
      </div>

      <DashboardSection title="Notes">
        <ActivityTable items={data.notes} />
      </DashboardSection>
    </div>
  )
}
