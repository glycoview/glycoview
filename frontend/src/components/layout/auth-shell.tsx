import type { ReactNode } from "react"

export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: string
  children: ReactNode
}) {
  return (
    <div className="min-h-screen bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(244,247,251,0.96))]">
      <div className="mx-auto flex min-h-screen max-w-7xl items-center px-4 py-10 sm:px-6 lg:px-8">
        <div className="grid w-full overflow-hidden rounded-[32px] border border-border/80 bg-white/94 shadow-[0_24px_80px_rgba(15,23,42,0.08)] lg:grid-cols-[minmax(320px,0.92fr)_minmax(420px,1.08fr)]">
          <section className="border-b border-border/80 bg-slate-100/85 px-8 py-10 text-slate-900 lg:border-b-0 lg:border-r">
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-900 text-lg font-extrabold text-white">
                B
              </div>
              <div>
                <div className="text-xl font-semibold tracking-tight">GlycoView</div>
                <div className="text-sm text-slate-500">Clinical dashboard</div>
              </div>
            </div>

            <div className="mt-14 max-w-sm">
              <div className="text-xs font-semibold uppercase tracking-[0.26em] text-slate-500">Access</div>
              <h1 className="mt-4 text-4xl font-semibold tracking-tight text-slate-950">{title}</h1>
              {subtitle ? <p className="mt-4 text-base leading-7 text-slate-600">{subtitle}</p> : null}
            </div>

            <div className="mt-14 space-y-6">
              <AuthInfo label="Role-based access" value="Admin and doctor accounts" />
              <AuthInfo label="Nightscout compatibility" value="API keys remain unchanged" />
              <AuthInfo label="Self-hosted setup" value="First admin created on initial launch" />
            </div>
          </section>

          <section className="flex items-center px-6 py-8 sm:px-10 lg:px-12 lg:py-12">
            <div className="mx-auto w-full max-w-md">{children}</div>
          </section>
        </div>
      </div>
    </div>
  )
}

function AuthInfo({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-l-2 border-slate-300 pl-4">
      <div className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500">{label}</div>
      <div className="mt-2 text-sm text-slate-700">{value}</div>
    </div>
  )
}
