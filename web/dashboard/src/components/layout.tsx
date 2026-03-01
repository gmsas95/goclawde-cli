import { 
  LayoutDashboard, MessageSquare, Settings, User, Wrench, FileText,
  Brain, Cpu, Sparkles, Bell
} from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Chat', href: '/chat', icon: MessageSquare },
  { name: 'Skills', href: '/skills', icon: Wrench },
  { name: 'Memory', href: '/memory', icon: Brain },
  { name: 'Jobs', href: '/jobs', icon: Cpu },
  { name: 'Persona', href: '/persona', icon: User },
  { name: 'Config', href: '/config', icon: Settings },
  { name: 'Logs', href: '/logs', icon: FileText },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()

  return (
    <div className="min-h-screen bg-background">
      {/* Sidebar */}
      <aside
        className={cn(
          "fixed top-0 left-0 z-40 h-screen w-64 border-r border-border bg-card",
          "hidden lg:block"
        )}
      >
        <div className="h-full flex flex-col">
          {/* Logo */}
          <div className="p-5 border-b border-border">
            <Link to="/" className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary">
                <Sparkles className="h-5 w-5 text-primary-foreground" />
              </div>
              <span className="font-semibold text-lg">Myrai</span>
            </Link>
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-3 space-y-1">
            <div className="mb-3 px-2">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Main</p>
            </div>
            
            {navigation.slice(0, 5).map((item) => {
              const isActive = location.pathname === item.href
              
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                    isActive
                      ? "bg-primary/10 text-primary"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground"
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  {item.name}
                </Link>
              )
            })}
            
            <div className="mt-6 mb-3 px-2">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">System</p>
            </div>
            
            {navigation.slice(5).map((item) => {
              const isActive = location.pathname === item.href
              
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                    isActive
                      ? "bg-primary/10 text-primary"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground"
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  {item.name}
                </Link>
              )
            })}
          </nav>

          {/* Footer */}
          <div className="p-4 border-t border-border">
            <div className="text-xs text-muted-foreground">
              <p>Myrai v2.0</p>
              <p className="mt-0.5">AI Assistant Platform</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:ml-64">
        {/* Top bar */}
        <header className="sticky top-0 z-30 border-b border-border bg-background/95 backdrop-blur-sm">
          <div className="flex h-14 items-center justify-between px-6">
            <h2 className="text-base font-medium">
              {navigation.find(n => n.href === location.pathname)?.name || 'Dashboard'}
            </h2>
            
            <div className="flex items-center gap-3">
              <Button variant="ghost" size="icon" className="relative">
                <Bell className="h-4 w-4" />
                <span className="absolute -top-0.5 -right-0.5 h-4 w-4 rounded-full bg-destructive text-[10px] font-medium text-destructive-foreground flex items-center justify-center">
                  3
                </span>
              </Button>
              
              <div className="h-4 w-px bg-border" />
              
              <div className="flex items-center gap-2">
                <div className="h-7 w-7 rounded-full bg-muted flex items-center justify-center">
                  <span className="text-xs font-medium">A</span>
                </div>
                <span className="text-sm text-muted-foreground hidden sm:inline">Admin</span>
              </div>
            </div>
          </div>
        </header>

        <main className="p-6">
          {children}
        </main>
      </div>
    </div>
  )
}
