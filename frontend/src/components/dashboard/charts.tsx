import type { GlucosePoint, TrendBucket } from "@/types"
import { clamp } from "@/lib/utils"

function scaleY(value: number, min: number, max: number, top: number, bottom: number) {
  const safe = clamp(value, min, max)
  const ratio = (safe - min) / (max - min || 1)
  return bottom - ratio * (bottom - top)
}

function linePoints(points: GlucosePoint[], width: number, height: number) {
  if (points.length < 2) return ""
  const minX = points[0].at
  const maxX = points[points.length - 1].at || minX + 1
  return points
    .map((point) => {
      const x = 24 + ((point.at - minX) / (maxX - minX || 1)) * (width - 48)
      const y = scaleY(point.value, 54, 280, 26, height - 26)
      return `${x},${y}`
    })
    .join(" ")
}

export function GlucoseSparkline({ points, tall = false }: { points: GlucosePoint[]; tall?: boolean }) {
  if (points.length < 2) {
    return <div className="flex h-52 items-center justify-center text-sm text-slate-500">Not enough glucose data</div>
  }

  const width = 820
  const height = tall ? 320 : 240
  const polyline = linePoints(points, width, height)
  const gridValues = [70, 100, 140, 180, 250]

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full">
      <rect x="12" y="12" width={width - 24} height={height - 24} rx="18" fill="#fbfcfe" />
      {gridValues.map((mark) => {
        const y = scaleY(mark, 54, 280, 26, height - 26)
        const stroke = mark === 70 ? "#fb7185" : mark === 180 ? "#f59e0b" : "#d6dee8"
        return (
          <line
            key={mark}
            x1="24"
            x2={width - 24}
            y1={y}
            y2={y}
            stroke={stroke}
            strokeDasharray="6 6"
            strokeWidth="1.4"
          />
        )
      })}
      <polyline points={polyline} fill="none" stroke="#1e3a5f" strokeWidth="4" strokeLinecap="round" />
    </svg>
  )
}

export function AgpChart({ buckets }: { buckets: TrendBucket[] }) {
  const active = buckets.some((bucket) => bucket.points > 0)
  if (!active) {
    return <div className="flex h-72 items-center justify-center text-sm text-slate-500">Not enough data for AGP</div>
  }

  const width = 920
  const height = 340
  const band90: string[] = []
  const band75: string[] = []
  const median: string[] = []

  buckets.forEach((bucket, index) => {
    const x = 28 + (index / 23) * (width - 56)
    band90.push(`${x},${scaleY(bucket.p10, 54, 280, 26, height - 26)}`)
    band75.push(`${x},${scaleY(bucket.p25, 54, 280, 26, height - 26)}`)
    median.push(`${x},${scaleY(bucket.p50, 54, 280, 26, height - 26)}`)
  })

  for (let index = buckets.length - 1; index >= 0; index -= 1) {
    const bucket = buckets[index]
    const x = 28 + (index / 23) * (width - 56)
    band90.push(`${x},${scaleY(bucket.p90, 54, 280, 26, height - 26)}`)
    band75.push(`${x},${scaleY(bucket.p75, 54, 280, 26, height - 26)}`)
  }

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full">
      <rect x="12" y="12" width={width - 24} height={height - 24} rx="18" fill="#fbfcfe" />
      <polygon points={band90.join(" ")} fill="rgba(125, 167, 214, 0.22)" />
      <polygon points={band75.join(" ")} fill="rgba(79, 120, 168, 0.24)" />
      <polyline points={median.join(" ")} fill="none" stroke="#1e3a5f" strokeWidth="3.5" />
      {[0, 3, 6, 9, 12, 15, 18, 21].map((hour) => (
        <text
          key={hour}
          x={28 + (hour / 23) * (width - 56) - 12}
          y={height - 10}
          fontSize="11"
          fill="#64748b"
        >
          {`${String(hour).padStart(2, "0")}:00`}
        </text>
      ))}
    </svg>
  )
}
