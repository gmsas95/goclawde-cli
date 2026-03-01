import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { 
  Activity, Bot, Server, Settings, TrendingUp,
  MessageSquare, Cpu, Shield, ArrowRight, Zap, Brain
} from 'lucide-react'
import { statusApi, skillsApi, activityApi } from '@/lib/api'
import { useStore } from '@/stores/app'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Link } from 'react-router-dom'

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

  const { data: activities = [] } = useQuery({
    queryKey: ['activity'],
    queryFn: activityApi.list,
  })

  useEffect(() => {
    if (status) {
      setStatus(status)
    }
  }, [status, setStatus])

  if (isLoading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex items-center gap-3 text-muted-foreground">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-primary" />
          <span className="text-sm">Loading dashboard...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">Overview of your AI assistant</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 rounded-full bg-accent/10 px-3 py-1.5">
            <div className="h-2 w-2 rounded-full bg-accent animate-pulse" />
            <span className="text-xs font-medium text-accent">System Online</span>
          </div>
          <span className="text-xs text-muted-foreground">v{status?.version}</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">System Status</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Online</div>
            <p className="text-xs text-muted-foreground mt-1">{status?.uptime || 'Running'}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Active Skills</CardTitle>
            <Settings className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{skills?.length || status?.skills || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Available capabilities</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">LLM Provider</CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold capitalize">{status?.llm.provider || 'N/A'}</div>
            <p className="text-xs text-muted-foreground mt-1 truncate">{status?.llm.model || 'No model configured'}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Active Channels</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status?.channels.telegram ? 1 : 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Connected platforms</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      <div className="grid gap-6 md:grid-cols-7">
        {/* Activity */}
        <Card className="md:col-span-4">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Recent Activity
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {activities.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-8">No recent activity</p>
              ) : (
                activities.slice(0, 5).map((item: any, i: number) => {
                  // Map activity types to icons and colors
                  const getIcon = (type: string) => {
                    switch (type) {
                      case 'conversation': return MessageSquare
                      case 'memory': return Brain
                      case 'task': return Cpu
                      default: return Activity
                    }
                  }
                  const getColor = (type: string) => {
                    switch (type) {
                      case 'conversation': return 'text-primary'
                      case 'memory': return 'text-accent'
                      case 'task': return 'text-muted-foreground'
                      default: return 'text-muted-foreground'
                    }
                  }
                  const Icon = getIcon(item.type)
                  return (
                    <div key={i} className="flex items-center gap-3">
                      <div className={`p-2 rounded-md bg-muted ${getColor(item.type)}`}>
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="flex-1">
                        <p className="text-sm">{item.text}</p>
                        <p className="text-xs text-muted-foreground">{item.time}</p>
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card className="md:col-span-3">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="h-4 w-4" />
              Quick Actions
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <Link to="/skills">
              <Button variant="outline" className="w-full justify-between">
                <span className="flex items-center gap-2">
                  <Settings className="h-4 w-4" />
                  Manage Skills
                </span>
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link to="/config">
              <Button variant="outline" className="w-full justify-between">
                <span className="flex items-center gap-2">
                  <MessageSquare className="h-4 w-4" />
                  Configure Channels
                </span>
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link to="/logs">
              <Button variant="outline" className="w-full justify-between">
                <span className="flex items-center gap-2">
                  <Shield className="h-4 w-4" />
                  View Logs
                </span>
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>

      {/* Connected Channels */}
      {status?.channels.telegram && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              Connected Channels
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between p-4 rounded-lg bg-muted">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-md bg-primary/10">
                  <MessageSquare className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="font-medium">Telegram</p>
                  <p className="text-sm text-muted-foreground">Bot connected and active</p>
                </div>
              </div>
              <div className="h-2.5 w-2.5 rounded-full bg-accent" />
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
