import { formatRelativeTime, formatValue } from "@/lib/utils"
import type { ActivityItem, DailySummary, DeviceCard, EventPoint, SchedulePoint } from "@/types"

export function ActivityTable({ items }: { items: ActivityItem[] }) {
  if (!items.length) return <EmptyMessage text="No recent activity in this window." />
  return (
    <div className="divide-y divide-border">
      {items.map((item) => (
        <div
          key={`${item.kind}-${item.at}-${item.title}`}
          className="flex flex-col gap-2 py-4 lg:grid lg:grid-cols-[minmax(0,1.1fr)_minmax(0,2fr)_auto] lg:items-start lg:gap-4"
        >
          <div className="text-sm font-semibold text-slate-900">{item.title}</div>
          <div className="text-sm text-slate-600">{item.detail}</div>
          <div className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500 lg:text-right">
            {formatRelativeTime(item.at)}
          </div>
        </div>
      ))}
    </div>
  )
}

export function DeviceFeedTable({ cards }: { cards: DeviceCard[] }) {
  if (!cards.length) return <EmptyMessage text="No device status records found." />
  return (
    <div className="divide-y divide-border">
      {cards.map((card) => (
        <div
          key={card.name}
          className="grid gap-4 py-4 md:grid-cols-2 2xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)]"
        >
          <div>
            <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">{card.kind}</div>
            <div className="mt-1 text-base font-semibold text-slate-950">{card.name}</div>
            <div className="mt-1 text-sm text-slate-600">{card.status}</div>
          </div>
          <DetailCell label="Connection" value={card.connection || "Connected"} />
          <DetailCell label="Last seen" value={formatRelativeTime(card.lastSeen)} />
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-1">
              <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Feed</div>
              <div className="text-sm text-slate-700">
                {[card.battery ? `Battery ${card.battery}%` : "", card.reservoir ? `Reservoir ${card.reservoir}` : ""]
                  .filter(Boolean)
                  .join(" · ") || "Monitoring"}
              </div>
            </div>
            {card.badge ? (
              <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                {card.badge}
              </div>
            ) : null}
          </div>
        </div>
      ))}
    </div>
  )
}

export function EventTable({
  events,
  unit,
  emptyText,
}: {
  events: EventPoint[]
  unit: string
  emptyText: string
}) {
  if (!events.length) return <EmptyMessage text={emptyText} />
  return (
    <div className="divide-y divide-border">
      {events.map((event) => (
        <div
          key={`${event.kind}-${event.at}-${event.value}`}
          className="flex flex-col gap-1 py-3 sm:grid sm:grid-cols-[minmax(0,1.3fr)_auto_auto] sm:items-start sm:gap-3"
        >
          <div>
            <div className="text-sm font-semibold text-slate-900">{event.subtitle || event.label}</div>
            <div className="text-xs uppercase tracking-[0.2em] text-slate-500">
              {new Date(event.at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
            </div>
          </div>
          <div className="text-sm text-slate-500">{event.kind}</div>
          <div className="text-sm font-semibold text-slate-950">{formatValue(event.value, unit)}</div>
        </div>
      ))}
    </div>
  )
}

export function ScheduleTable({ rows }: { rows: SchedulePoint[] }) {
  if (!rows.length) return <EmptyMessage text="No schedule entries available." />
  return (
    <div className="divide-y divide-border">
      {rows.map((row) => (
        <div key={`${row.time}-${row.value}`} className="flex items-center justify-between gap-4 py-3">
          <div className="text-sm font-semibold text-slate-900">{row.time}</div>
          <div className="text-sm text-slate-600">{row.value}</div>
        </div>
      ))}
    </div>
  )
}

export function DailySummaryTable({ rows }: { rows: DailySummary[] }) {
  if (!rows.length) return <EmptyMessage text="No multi-day summary data available." />
  return (
    <div className="space-y-3">
      <div className="space-y-3 md:hidden">
        {rows.map((row) => (
          <div key={row.date} className="rounded-2xl border border-border/80 bg-slate-50/70 px-4 py-4">
            <div className="text-sm font-semibold text-slate-900">
              {new Date(row.date).toLocaleDateString([], { month: "short", day: "2-digit" })}
            </div>
            <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
              <SummaryCell label="Avg" value={`${Math.round(row.avgGlucose)} mg/dL`} />
              <SummaryCell label="Carbs" value={`${Math.round(row.carbs)} g`} />
              <SummaryCell label="Insulin" value={`${row.insulin.toFixed(1)} U`} />
              <SummaryCell label="TIR" value={`${Math.round(row.tir)}%`} />
            </div>
          </div>
        ))}
      </div>

      <div className="hidden md:block">
        <div className="grid grid-cols-[1.2fr_repeat(4,minmax(0,1fr))] gap-3 border-b border-border pb-3 text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500">
          <span>Day</span>
          <span>Avg</span>
          <span>Carbs</span>
          <span>Insulin</span>
          <span>TIR</span>
        </div>
        <div className="divide-y divide-border">
          {rows.map((row) => (
            <div key={row.date} className="grid grid-cols-[1.2fr_repeat(4,minmax(0,1fr))] gap-3 py-3 text-sm">
              <span className="font-semibold text-slate-900">
                {new Date(row.date).toLocaleDateString([], { month: "short", day: "2-digit" })}
              </span>
              <span className="text-slate-600">{Math.round(row.avgGlucose)} mg/dL</span>
              <span className="text-slate-600">{Math.round(row.carbs)} g</span>
              <span className="text-slate-600">{row.insulin.toFixed(1)} U</span>
              <span className="font-semibold text-slate-900">{Math.round(row.tir)}%</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function DetailCell({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">{label}</div>
      <div className="mt-1 text-sm text-slate-700">{value}</div>
    </div>
  )
}

function EmptyMessage({ text }: { text: string }) {
  return <div className="py-10 text-sm text-slate-500">{text}</div>
}

function SummaryCell({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-500">{label}</div>
      <div className="mt-1 text-sm font-medium text-slate-900">{value}</div>
    </div>
  )
}
