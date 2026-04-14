import { DashboardSection } from "@/components/dashboard/section"
import { MetricStrip } from "@/components/dashboard/metric-strip"
import { GlucoseSparkline } from "@/components/dashboard/charts"
import { TimeInRange } from "@/components/dashboard/time-in-range"
import { ActivityTable, DeviceFeedTable } from "@/components/dashboard/lists"
import { Skeleton } from "@/components/ui/skeleton"
import { useApiResource } from "@/lib/api"
import { formatMinutes } from "@/lib/utils"
import type { OverviewResponse } from "@/types"

export function OverviewPage({ token }: { token: string }) {
  const { data, loading, error } = useApiResource<OverviewResponse>("/app/api/overview", token)

  if (loading) return <OverviewSkeleton />
  if (error || !data) return <ErrorState text={error?.message ?? "Failed to load overview"} />

  return (
    <div className="space-y-6">
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.85fr)]">
        <DashboardSection
          title="Current glucose state"
          subtitle={data.patientName || undefined}
        >
          <div className="grid gap-6 xl:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
            <div className="space-y-5 border-b border-border pb-5 xl:border-b-0 xl:border-r xl:pb-0 xl:pr-6">
              <div>
                <div className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-500">
                  {data.current.label}
                </div>
                <div className="mt-3 text-6xl font-semibold tracking-tight text-slate-950">
                  {data.current.value}
                </div>
                <div className="mt-2 text-sm text-slate-500">{data.current.detail}</div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="border-l-2 border-sky-500/70 px-4 py-1">
                  <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                    Review window
                  </div>
                  <div className="mt-2 text-2xl font-semibold tracking-tight text-slate-950">
                    {data.subtitle}
                  </div>
                </div>
                <div className="border-l-2 border-teal-500/70 px-4 py-1">
                  <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                    Tight range
                  </div>
                  <div className="mt-2 text-2xl font-semibold tracking-tight text-slate-950">
                    {Math.round(data.narrowRange.percent)}%
                  </div>
                  <div className="mt-1 text-sm text-slate-500">{formatMinutes(data.narrowRange.minutes)}</div>
                </div>
              </div>
            </div>

            <GlucoseSparkline points={data.sparkline} />
          </div>
        </DashboardSection>

        <DashboardSection title="Range distribution">
          <TimeInRange bands={[...data.timeInRange, data.narrowRange]} />
        </DashboardSection>
      </div>

      <MetricStrip metrics={data.metrics} />

      <div className="grid gap-6 xl:grid-cols-2">
        <DashboardSection title="Connected devices">
          <DeviceFeedTable cards={data.devices} />
        </DashboardSection>
        <DashboardSection title="Clinical activity">
          <ActivityTable items={data.activity} />
        </DashboardSection>
      </div>
    </div>
  )
}

function OverviewSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.85fr)]">
        <Skeleton className="h-[360px] rounded-2xl" />
        <Skeleton className="h-[360px] rounded-2xl" />
      </div>
      <div className="grid gap-3 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-24 rounded-2xl" />
        ))}
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        <Skeleton className="h-[320px] rounded-2xl" />
        <Skeleton className="h-[320px] rounded-2xl" />
      </div>
    </div>
  )
}

function ErrorState({ text }: { text: string }) {
  return (
    <div className="rounded-2xl border border-amber-200 bg-amber-50 px-6 py-5">
      <p className="text-sm text-amber-900">{text}</p>
    </div>
  )
}
