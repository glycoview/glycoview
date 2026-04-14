import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export function DashboardSection({
  title,
  subtitle,
  className,
  children,
}: {
  title: string
  subtitle?: string
  className?: string
  children: ReactNode
}) {
  return (
    <section
      className={cn(
        "dashboard-panel",
        className,
      )}
    >
      <header className="dashboard-panel-header">
        <h2 className="dashboard-title">{title}</h2>
        {subtitle ? <p className="dashboard-subtle mt-1">{subtitle}</p> : null}
      </header>
      <div className="dashboard-panel-body">{children}</div>
    </section>
  )
}
