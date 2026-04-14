import type { TimeInRangeBand } from "@/types"
import { formatMinutes } from "@/lib/utils"

const accentMap: Record<string, string> = {
  rose: "bg-rose-400",
  pink: "bg-rose-300",
  blue: "bg-sky-600",
  amber: "bg-amber-300",
  orange: "bg-amber-500",
  cyan: "bg-teal-500",
}

export function TimeInRange({ bands }: { bands: TimeInRangeBand[] }) {
  return (
    <div className="space-y-4">
      {bands.map((band) => (
        <div key={`${band.label}-${band.range}`} className="space-y-2">
          <div className="flex items-end justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-slate-900">{band.label}</div>
              <div className="text-xs uppercase tracking-[0.2em] text-slate-500">{band.range} mg/dL</div>
            </div>
            <div className="text-right">
              <div className="text-lg font-semibold text-slate-950">{Math.round(band.percent)}%</div>
              <div className="text-xs text-slate-500">{formatMinutes(band.minutes)}</div>
            </div>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-slate-100">
            <div
              className={`h-full rounded-full ${accentMap[band.accent] ?? "bg-slate-500"}`}
              style={{ width: `${band.percent}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}
