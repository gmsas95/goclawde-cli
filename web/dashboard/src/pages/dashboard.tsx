import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { 
  Activity, Bot, Server, Settings, Zap, TrendingUp, 
  MessageSquare, Brain, Cpu, Shield, Clock 
} from 'lucide-react'
import { statusApi, skillsApi } from '@/lib/api'
import { useStore } from '@/stores/app'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { StatsCard } from '@/components/stats-card'

export function Dashboard() {
  const { setStatus } = useStore()
  
  const { data: status, isLoading } = useQuery({
    queryKey: ['status'],
    queryFn: statusApi.get,
    refetchInterval: 5000,
  })

  const { data: skills } = useQuery({
    queryKey: ['skills'],
    queryFn: skillsApi.list,
  })

  useEffect(() => {
    if (status) {
      setStatus(status)
    }
  }, [status, setStatus])

  if (isLoading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="relative">
          <div className="h-16 w-16 animate-spin rounded-full border-4 border-white/10 border-t-cyan-500" />
          <div className="absolute inset-0 h-16 w-16 animate-ping rounded-full border-4 border-cyan-500/30" />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Hero Section */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-violet-600/20 via-cyan-600/20 to-blue-600/20 p-8 backdrop-blur-xl border border-white/10">
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 shadow-lg shadow-cyan-500/30">
              <Brain className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-4xl font-bold bg-gradient-to-r from-white to-white/70 bg-clip-text text-transparent">
                Myrai Dashboard
              </h1>
              <p className="text-white/60">Your personal AI assistant control center</p>
            </div>
          </div>
          
          <div className="mt-6 flex items-center gap-4">
            <div className="flex items-center gap-2 rounded-full bg-emerald-500/20 px-4 py-2 border border-emerald-500/30">
              <div className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
              <span className="text-sm font-medium text-emerald-400">System Online</span>
            </div>
            <span className="text-white/40">Version {status?.version}</span>
          </div>
        </div>
        
        {/* Decorative elements */}
        <div className="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-cyan-500/20 blur-3xl" />
        <div className="absolute -bottom-20 -left-20 h-64 w-64 rounded-full bg-violet-500/20 blur-3xl" />
      </div>

      {/* Stats Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <StatsCard
          title="System Status"
          value="Online"
          description={`Uptime: ${status?.uptime || 'Running'}`}
          icon={Server}
          color="emerald"
          trend={{ value: 99.9, positive: true }}
        />
        
        <StatsCard
          title="Active Skills"
          value={skills?.length || status?.skills || 0}
          description="Available capabilities"
          icon={Settings}
          color="cyan"
        />
        
        <StatsCard
          title="LLM Provider"
          value={status?.llm.provider || 'N/A'}
          description={status?.llm.model || 'No model configured'}
          icon={Bot}
          color="violet"
        />
        
        <StatsCard
          title="Channels"
          value={
            [status?.channels.telegram, status?.channels.discord].filter(Boolean).length
          }
          description="Active integrations"
          icon={MessageSquare}
          color="amber"
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Activity Chart Placeholder */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Activity className="h-5 w-5 text-cyan-400" />
                  Activity Overview
                </CardTitle>
                <CardDescription>Real-time system metrics</CardDescription>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm">24h</Button>
                <Button variant="outline" size="sm">7d</Button>
                <Button variant="outline" size="sm">30d</Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] rounded-xl bg-white/5 flex items-center justify-center border border-white/10">
              <div className="text-center">
                <TrendingUp className="h-12 w-12 text-white/20 mx-auto mb-4" />
                <p className="text-white/40">Activity charts coming soon</p>
                <p className="text-sm text-white/30 mt-2">Visualize your AI usage patterns</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="h-5 w-5 text-amber-400" />
              Quick Actions
            </CardTitle>
            <CardDescription>Common operations</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button variant="glass" className="w-full justify-start">
              <Brain className="mr-2 h-4 w-4" />
              Test AI Connection
            </Button>
            <Button variant="glass" className="w-full justify-start">
              <Cpu className="mr-2 h-4 w-4" />
              System Diagnostics
            </Button>
            <Button variant="glass" className="w-full justify-start">
              <Shield className="mr-2 h-4 w-4" />
              Security Check
            </Button>
            <Button variant="glass" className="w-full justify-start">
              <Clock className="mr-2 h-4 w-4" />
              View Job Queue
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Bottom Section */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5 text-violet-400" />
              Connected Channels
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {status?.channels.telegram && (
                <div className="flex items-center justify-between rounded-xl bg-white/5 p-4 border border-white/10">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/20">
                      <MessageSquare className="h-5 w-5 text-blue-400" />
                    </div>
                    <div>
                      <p className="font-medium text-white">Telegram</p>
                      <p className="text-sm text-white/50">Bot connected</p>
                    </div>
                  </div>
                  <div className="h-2 w-2 rounded-full bg-emerald-400" />
                </div>
              )}
              
              {status?.channels.discord && (
                <div className="flex items-center justify-between rounded-xl bg-white/5 p-4 border border-white/10">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/20">
                      <MessageSquare className="h-5 w-5 text-indigo-400" />
                    </div>
                    <div>
                      <p className="font-medium text-white">Discord</p>
                      <p className="text-sm text-white/50">Bot connected</p>
                    </div>
                  </div>
                  <div className="h-2 w-2 rounded-full bg-emerald-400" />
                </div>
              )}
              
              {!status?.channels.telegram && !status?.channels.discord && (
                <div className="text-center py-8 text-white/40">
                  <MessageSquare className="h-12 w-12 mx-auto mb-3 opacity-30" />
                  <p>No channels configured</p>
                  <p className="text-sm mt-1">Connect Telegram or Discord in settings</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5 text-cyan-400" />
              Skill Overview
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3">
              {skills?.slice(0, 6).map((skill) => (
                <div
                  key={skill.name}
                  className="flex items-center gap-3 rounded-xl bg-white/5 p-3 border border-white/10 hover:bg-white/10 transition-colors"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/20">
                    <Settings className="h-4 w-4 text-cyan-400" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-white truncate capitalize">{skill.name}</p>
                    <p className="text-xs text-white/50">{skill.tools} tools</p>
                  </div>
                  <div className={cn(
                    'h-2 w-2 rounded-full',
                    skill.enabled ? 'bg-emerald-400' : 'bg-white/20'
                  )} />
                </div>
              ))}
            </div>
            
            {skills && skills.length > 6 && (
              <p className="text-center text-sm text-white/40 mt-4">
                +{skills.length - 6} more skills
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function cn(...inputs: (string | undefined | null | false)[]) {
  return inputs.filter(Boolean).join(' ')
}
