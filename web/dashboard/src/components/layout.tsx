import { LayoutDashboard, MessageSquare, Settings, User, Wrench, FileText, Menu, X } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { useStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Chat', href: '/chat', icon: MessageSquare },
  { name: 'Config', href: '/config', icon: Settings },
  { name: 'Persona', href: '/persona', icon: User },
  { name: 'Skills', href: '/skills', icon: Wrench },
  { name: 'Logs', href: '/logs', icon: FileText },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const { sidebarOpen, toggleSidebar } = useStore()
  const location = useLocation()

  return (
    <div className="min-h-screen bg-background">
      {/* Mobile sidebar toggle */}
      <div className="lg:hidden fixed top-4 left-4 z-50">
        <Button
          variant="outline"
          size="icon"
          onClick={toggleSidebar}
        >
          {sidebarOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
        </Button>
      </div>

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed top-0 left-0 z-40 h-screen transition-transform bg-card border-r",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
          "w-64 lg:translate-x-0"
        )}
      >
        <div className="h-full flex flex-col">
          {/* Logo */}
          <div className="p-6 border-b">
            <Link to="/" className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
                <span className="text-primary-foreground font-bold text-lg">M</span>
              </div>
              <span className="font-semibold text-lg">Myrai</span>
            </Link>
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-4 space-y-1">
            {navigation.map((item) => {
              const isActive = location.pathname === item.href
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  {item.name}
                </Link>
              )
            })}
          </nav>

          {/* Footer */}
          <div className="p-4 border-t">
            <div className="text-xs text-muted-foreground">
              <p>Myrai 2.0</p>
              <p className="mt-1">未来はここにある</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main
        className={cn(
          "transition-all duration-300",
          sidebarOpen ? "lg:ml-64" : ""
        )}
      >
        <div className="p-8">
          {children}
        </div>
      </main>
    </div>
  )
}
