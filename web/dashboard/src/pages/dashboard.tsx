import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity, Bot, Server, Settings, Zap } from 'lucide-react'
import { statusApi } from '@/lib/api'
import { useStore } from '@/stores/app'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

export function Dashboard() {
  const { setStatus } = useStore()
  
  const { data: status, isLoading } = useQuery({
    queryKey: ['status'],
    queryFn: statusApi.get,
    refetchInterval: 5000, // Refetch every 5 seconds
  })

  useEffect(() => {
    if (status) {
      setStatus(status)
    }
  }, [status, setStatus])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">Overview of your Myrai instance</p>
      </div>

      {/* Status Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Status</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">Online</div>
            <p className="text-xs text-muted-foreground">
              {status?.uptime || '0s'} uptime
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">LLM Provider</CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status?.llm.provider || 'N/A'}</div>
            <p className="text-xs text-muted-foreground">
              {status?.llm.model || 'No model'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Skills</CardTitle>
            <Settings className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status?.skills || 0}</div>
            <p className="text-xs text-muted-foreground">
              Active capabilities
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Channels</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              {status?.channels.telegram && (
                <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-1 text-xs font-medium text-green-700">
                  Telegram
                </span>
              )}
              {status?.channels.discord && (
                <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-1 text-xs font-medium text-green-700">
                  Discord
                </span>
              )}
              {!status?.channels.telegram && !status?.channels.discord && (
                <span className="text-xs text-muted-foreground">None configured</span>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
          <CardDescription>Common tasks and operations</CardDescription>
        </CardHeader>
        <CardContent className="flex gap-4">
          <Button>
            <Zap className="mr-2 h-4 w-4" />
            Restart Server
          </Button>
          <Button variant="outline">
            View Logs
          </Button>
          <Button variant="outline">
            Check Health
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
