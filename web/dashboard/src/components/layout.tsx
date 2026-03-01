import { 
  LayoutDashboard, MessageSquare, Settings, User, Wrench, FileText, 
  Menu, X, Brain, LogOut, Bell, Shield, Cpu, Sparkles 
} from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { useStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard, color: 'cyan' },
  { name: 'Chat', href: '/chat', icon: MessageSquare, color: 'violet' },
  { name: 'Skills', href: '/skills', icon: Wrench, color: 'amber' },
  { name: 'Memory', href: '/memory', icon: Brain, color: 'rose' },
  { name: 'Jobs', href: '/jobs', icon: Cpu, color: 'emerald' },
  { name: 'Persona', href: '/persona', icon: User, color: 'pink' },
  { name: 'Config', href: '/config', icon: Settings, color: 'blue' },
  { name: 'Logs', href: '/logs', icon: FileText, color: 'slate' },
]

const colorClasses: Record<string, { bg: string; text: string; border: string }> = {
  cyan: { bg: 'bg-cyan-500/10', text: 'text-cyan-400', border: 'border-cyan-500/30' },
  violet: { bg: 'bg-violet-500/10', text: 'text-violet-400', border: 'border-violet-500/30' },
  amber: { bg: 'bg-amber-500/10', text: 'text-amber-400', border: 'border-amber-500/30' },
  rose: { bg: 'bg-rose-500/10', text: 'text-rose-400', border: 'border-rose-500/30' },
  emerald: { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/30' },
  pink: { bg: 'bg-pink-500/10', text: 'text-pink-400', border: 'border-pink-500/30' },
  blue: { bg: 'bg-blue-500/10', text: 'text-blue-400', border: 'border-blue-500/30' },
  slate: { bg: 'bg-slate-500/10', text: 'text-slate-400', border: 'border-slate-500/30' },
}

export function Layout({ children }: { children: React.ReactNode }) {
  const { sidebarOpen, toggleSidebar } = useStore()
  const location = useLocation()

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Mobile sidebar toggle */}
      <div className="lg:hidden fixed top-4 left-4 z-50">
        <Button
          variant="glass"
          size="icon"
          onClick={toggleSidebar}
          className="backdrop-blur-xl"
        >
          {sidebarOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
        </Button>
      </div>

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed top-0 left-0 z-40 h-screen transition-all duration-300",
          "bg-slate-950/80 backdrop-blur-2xl border-r border-white/10",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
          "w-72 lg:translate-x-0"
        )}
      >
        <div className="h-full flex flex-col">
          {/* Logo */}
          <div className="p-6 border-b border-white/10">
            <Link to="/" className="flex items-center gap-3">
              <div className="relative">
                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 flex items-center justify-center shadow-lg shadow-cyan-500/30">
                  <Sparkles className="h-6 w-6 text-white" />
                </div>
                <div className="absolute -inset-1 rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 blur-lg opacity-50" />
              </div>
              <div>
                <span className="font-bold text-xl text-white">Myrai</span>
                <p className="text-xs text-white/50">未来はここにある</p>
              </div>
            </Link>
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
            <div className="mb-4 px-3">
              <p className="text-xs font-semibold text-white/30 uppercase tracking-wider">Main</p>
            </div>
            
            {navigation.slice(0, 5).map((item) => {
              const isActive = location.pathname === item.href
              const colors = colorClasses[item.color]
              
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-all duration-200 group",
                    isActive
                      ? cn(colors.bg, colors.text, 'border', colors.border)
                      : "text-white/60 hover:bg-white/5 hover:text-white"
                  )}
                >
                  <div className={cn(
                    "p-2 rounded-lg transition-colors",
                    isActive ? colors.bg : "bg-white/5 group-hover:bg-white/10"
                  )}>
                    <item.icon className={cn(
                      "h-4 w-4",
                      isActive ? colors.text : "text-white/60"
                    )} />
                  </div>
                  {item.name}
                  
                  {isActive && (
                    <div className={cn("ml-auto h-2 w-2 rounded-full", colors.text.replace('text-', 'bg-'))} />
                  )}
                </Link>
              )
            })}
            
            <div className="mt-6 mb-4 px-3">
              <p className="text-xs font-semibold text-white/30 uppercase tracking-wider">System</p>
            </div>
            
            {navigation.slice(5).map((item) => {
              const isActive = location.pathname === item.href
              const colors = colorClasses[item.color]
              
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-all duration-200 group",
                    isActive
                      ? cn(colors.bg, colors.text, 'border', colors.border)
                      : "text-white/60 hover:bg-white/5 hover:text-white"
                  )}
                >
                  <div className={cn(
                    "p-2 rounded-lg transition-colors",
                    isActive ? colors.bg : "bg-white/5 group-hover:bg-white/10"
                  )}>
                    <item.icon className={cn(
                      "h-4 w-4",
                      isActive ? colors.text : "text-white/60"
                    )} />
                  </div>
                  {item.name}
                </Link>
              )
            })}
          </nav>

          {/* Footer */}
          <div className="p-4 border-t border-white/10">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="h-8 w-8 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center">
                  <Shield className="h-4 w-4 text-white" />
                </div>
                <div className="text-xs">
                  <p className="text-white/70">System Secure</p>
                  <p className="text-white/40">Myrai 2.0</p>
                </div>
              </div>
              
              <Button variant="ghost" size="icon" className="text-white/40 hover:text-white">
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className={cn("transition-all duration-300", sidebarOpen ? "lg:ml-72" : "")}>
        {/* Top bar */}
        <header className="sticky top-0 z-30 border-b border-white/10 bg-slate-950/50 backdrop-blur-xl">
          <div className="flex h-16 items-center justify-between px-8">
            <h2 className="text-lg font-semibold text-white">
              {navigation.find(n => n.href === location.pathname)?.name || 'Dashboard'}
            </h2>
            
            <div className="flex items-center gap-4">
              <Button variant="ghost" size="icon" className="text-white/60 hover:text-white relative">
                <Bell className="h-5 w-5" />
                <span className="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-rose-500 text-[10px] font-bold text-white flex items-center justify-center">3</span>
              </Button>
              
              <div className="h-8 w-px bg-white/10" />
              
              <div className="flex items-center gap-3">
                <div className="h-8 w-8 rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
                  <span className="text-white font-bold text-sm">A</span>
                </div>
                <span className="text-sm text-white/70 hidden sm:inline">Admin</span>
              </div>
            </div>
          </div>
        </header>

        <main className="p-8">
          {children}
        </main>
      </div>
    </div>
  )
}
