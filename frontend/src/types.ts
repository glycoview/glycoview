export type Metric = {
  id?: string
  label: string
  value: string
  detail?: string
  accent?: string
  delta?: string
  warning?: boolean
}

export type TimeInRangeBand = {
  label: string
  range: string
  minutes: number
  percent: number
  accent: string
}

export type GlucosePoint = {
  at: number
  value: number
  direction?: string
}

export type EventPoint = {
  at: number
  label: string
  kind: string
  value: number
  subtitle?: string
}

export type DeviceCard = {
  name: string
  kind: string
  status: string
  lastSeen: number
  battery?: string
  reservoir?: string
  badge?: string
  connection?: string
  details?: Metric[]
}

export type ActivityItem = {
  at: number
  title: string
  detail: string
  kind: string
  accent?: string
  primary?: boolean
}

export type OverviewResponse = {
  generatedAt: number
  patientName: string
  subtitle: string
  current: Metric
  sparkline: GlucosePoint[]
  timeInRange: TimeInRangeBand[]
  narrowRange: TimeInRangeBand
  metrics: Metric[]
  devices: DeviceCard[]
  activity: ActivityItem[]
}

export type DailyResponse = {
  generatedAt: number
  patientName: string
  dateLabel: string
  rangeStart: number
  rangeEnd: number
  glucose: GlucosePoint[]
  carbs: EventPoint[]
  insulin: EventPoint[]
  basalProfile: EventPoint[]
  timeInRange: TimeInRangeBand[]
  metrics: Metric[]
  devices: DeviceCard[]
}

export type TrendBucket = {
  hour: number
  p10: number
  p25: number
  p50: number
  p75: number
  p90: number
  points: number
}

export type DailySummary = {
  day: string
  date: number
  avgGlucose: number
  carbs: number
  insulin: number
  tir: number
}

export type TrendsResponse = {
  generatedAt: number
  patientName: string
  rangeLabel: string
  days: number
  agp: TrendBucket[]
  timeInRange: TimeInRangeBand[]
  metrics: Metric[]
  daysSummary: DailySummary[]
}

export type SchedulePoint = {
  time: string
  value: string
  label?: string
}

export type ProfileResponse = {
  generatedAt: number
  patientName: string
  headline: string
  metrics: Metric[]
  basalSchedule: SchedulePoint[]
  carbRatios: SchedulePoint[]
  sensitivity: SchedulePoint[]
  targets: SchedulePoint[]
  notes: ActivityItem[]
}

export type DevicesResponse = {
  generatedAt: number
  patientName: string
  headline: string
  cards: DeviceCard[]
  metrics: Metric[]
  activity: ActivityItem[]
}

export type AppUser = {
  id: string
  username: string
  displayName: string
  role: "admin" | "doctor"
  active: boolean
  createdAt: string
  updatedAt: string
}

export type AuthStatus = {
  setupRequired: boolean
  authenticated: boolean
  user?: AppUser
}

export type TLSField = {
  key: string
  label: string
  placeholder?: string
  secret?: boolean
}

export type TLSProvider = {
  id: string
  label: string
  fields: TLSField[]
}

export type ApplianceTLSConfig = {
  domain: string
  email: string
  challengeType: "http-01" | "dns-01" | string
  provider?: string
  env?: Record<string, string>
  configuredAt?: string
  appliedAt?: string
}

export type ApplianceStatus = {
  service: string
  dockerManaged: boolean
  stackName: string
  stackFile: string
  stackEnvFile: string
  currentTag: string
  currentImage: string
  currentAgentTag: string
  currentAgentImage: string
  lastAction?: string
  lastMessage?: string
  lastActionAt?: string
  tls: ApplianceTLSConfig
}

export type UpdateCheckResponse = {
  currentTag: string
  latestTag?: string
  updateAvailable: boolean
  releaseUrl?: string
  checkedAt?: string
  source: string
}

export type ApplianceActionResponse = {
  status: string
  message: string
  currentTag?: string
  currentAgentTag?: string
  appliedAt?: string
}
