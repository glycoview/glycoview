import { Activity, ChartSpline, HeartPulse, LogOut, MonitorCog, Settings, Shield, UserRound } from "lucide-react"
import { useMemo } from "react"
import { NavLink, Outlet, useLocation } from "react-router-dom"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import type { AppUser } from "@/types"

type AppShellProps = {
  user: AppUser
  onLogout: () => void
}

const baseNavItems = [
  { to: "/", label: "Overview", icon: Activity },
  { to: "/daily", label: "Daily", icon: ChartSpline },
  { to: "/trends", label: "Trends", icon: HeartPulse },
  { to: "/profile", label: "Profile", icon: UserRound },
  { to: "/devices", label: "Devices", icon: MonitorCog },
]

const pageMeta: Record<string, string> = {
  "/": "Overview",
  "/daily": "Daily",
	"/trends": "Trends",
  "/profile": "Profile",
	"/devices": "Devices",
  "/users": "Users",
  "/settings": "Settings",
}

export function AppShell({ user, onLogout }: AppShellProps) {
  const location = useLocation()
  const title = useMemo(() => pageMeta[location.pathname] ?? pageMeta["/"], [location.pathname])
  const navItems = useMemo(
    () =>
      user.role === "admin"
        ? [...baseNavItems, { to: "/users", label: "Users", icon: Shield }, { to: "/settings", label: "Settings", icon: Settings }]
        : baseNavItems,
    [user.role],
  )
  const userInitials = initials(user.displayName || user.username)

  return (
    <SidebarProvider defaultOpen className="bg-background text-foreground">
      <Sidebar collapsible="icon" className="border-r border-sidebar-border/80">
        <SidebarHeader className="px-3 py-4">
          <NavLink
            to="/"
            className="flex items-center gap-3 rounded-lg px-2 py-2 text-sidebar-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sidebar-primary text-sm font-semibold text-sidebar-primary-foreground">
              B
            </div>
            <div className="min-w-0 group-data-[collapsible=icon]:hidden">
              <div className="truncate text-base font-semibold tracking-tight">GlycoView</div>
              <div className="truncate text-xs text-muted-foreground">Clinical dashboard</div>
            </div>
          </NavLink>
        </SidebarHeader>

        <SidebarContent className="px-2">
          <SidebarGroup className="p-0">
            <SidebarGroupLabel className="px-2 text-[11px] font-semibold uppercase tracking-[0.18em]">
              Workspace
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems.map((item) => {
                  const Icon = item.icon
                  const isActive = item.to === "/" ? location.pathname === "/" : location.pathname.startsWith(item.to)

                  return (
                    <SidebarMenuItem key={item.to}>
                      <SidebarMenuButton asChild isActive={isActive} tooltip={item.label} className="h-10 rounded-lg px-3">
                        <NavLink to={item.to}>
                          <Icon />
                          <span>{item.label}</span>
                        </NavLink>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter className="px-3 pb-4 pt-2">
          <SidebarSeparator />
          <div className="flex items-center gap-3 px-2 py-3 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
            <Avatar size="sm">
              <AvatarFallback className="bg-sidebar-accent text-sidebar-accent-foreground">{userInitials}</AvatarFallback>
            </Avatar>
            <div className="min-w-0 group-data-[collapsible=icon]:hidden">
              <div className="truncate text-sm font-medium">{user.displayName}</div>
              <div className="truncate text-xs uppercase tracking-[0.16em] text-muted-foreground">{user.role}</div>
            </div>
          </div>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton onClick={onLogout} tooltip="Logout" className="h-10 rounded-lg px-3">
                <LogOut />
                <span>Logout</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>

      <SidebarInset className="min-h-screen bg-transparent">
        <header className="sticky top-0 z-20 border-b border-border/70 bg-background/90 backdrop-blur-sm">
          <div className="flex h-16 items-center gap-3 px-4 sm:px-6 xl:px-8">
            <SidebarTrigger className="text-muted-foreground" />
            <div className="min-w-0">
              <h1 className="truncate text-xl font-semibold tracking-tight text-foreground">{title}</h1>
            </div>
          </div>
        </header>

        <main className="dashboard-grid min-h-[calc(100vh-4rem)] px-4 py-5 sm:px-6 xl:px-8 xl:py-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}

function initials(value: string) {
  return value
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("")
}
