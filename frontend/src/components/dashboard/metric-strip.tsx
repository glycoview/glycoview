import type { Metric } from "@/types"

export function MetricStrip({ metrics }: { metrics: Metric[] }) {
  return (
    <div className="dashboard-panel">
      <div className="grid md:grid-cols-2 xl:grid-cols-4">
        {metrics.map((metric) => (
          <div
            key={metric.id ?? metric.label}
            className="border-b border-border/70 px-4 py-4 last:border-b-0 md:[&:nth-child(odd)]:border-r md:[&:nth-last-child(-n+2)]:border-b-0 xl:border-b-0 xl:border-r xl:[&:nth-child(2n)]:border-r xl:[&:last-child]:border-r-0"
          >
            <div className="dashboard-kicker">{metric.label}</div>
            <div className="mt-2 text-2xl font-semibold tracking-tight text-foreground">{metric.value}</div>
            {metric.detail ? <div className="dashboard-subtle mt-1">{metric.detail}</div> : null}
          </div>
        ))}
      </div>
    </div>
  )
}
