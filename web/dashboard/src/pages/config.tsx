import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Save, RefreshCw } from 'lucide-react'
import { configApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useStore } from '@/stores/app'
import type { Config } from '@/types'

export function Config() {
  const queryClient = useQueryClient()
  const { setConfig } = useStore()
  const [formData, setFormData] = useState({
    kimiApiKey: '',
    telegramToken: '',
    serverPort: 8080,
  })

  const { data: configData, isLoading } = useQuery<Config>({
    queryKey: ['config'],
    queryFn: configApi.get,
  })

  useEffect(() => {
    if (configData) {
      setConfig(configData)
      setFormData({
        kimiApiKey: configData.llm.providers.kimi?.api_key || '',
        telegramToken: configData.channels.telegram?.bot_token || '',
        serverPort: configData.server.port,
      })
    }
  }, [configData, setConfig])

  const updateMutation = useMutation({
    mutationFn: configApi.update,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      alert('Configuration updated successfully!')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate({
      llm: {
        default_provider: 'kimi',
        providers: {
          kimi: {
            api_key: formData.kimiApiKey,
            model: 'kimi-k2.5',
          },
        },
      },
      channels: {
        telegram: {
          enabled: !!formData.telegramToken,
          bot_token: formData.telegramToken,
        },
      },
      server: {
        port: formData.serverPort,
        address: '0.0.0.0',
      },
    } as Partial<Config>)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Configuration</h1>
        <p className="text-muted-foreground">Manage your Myrai settings</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* LLM Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>LLM Provider</CardTitle>
            <CardDescription>Configure your AI model settings</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Kimi API Key</label>
              <input
                type="password"
                value={formData.kimiApiKey}
                onChange={(e) => setFormData({ ...formData, kimiApiKey: e.target.value })}
                placeholder="sk-..."
                className="w-full px-3 py-2 border rounded-md bg-background"
              />
            </div>
          </CardContent>
        </Card>

        {/* Channels */}
        <Card>
          <CardHeader>
            <CardTitle>Channels</CardTitle>
            <CardDescription>Configure messaging platforms</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Telegram Bot Token</label>
              <input
                type="password"
                value={formData.telegramToken}
                onChange={(e) => setFormData({ ...formData, telegramToken: e.target.value })}
                placeholder="123456:ABC..."
                className="w-full px-3 py-2 border rounded-md bg-background"
              />
            </div>
          </CardContent>
        </Card>

        {/* Server */}
        <Card>
          <CardHeader>
            <CardTitle>Server</CardTitle>
            <CardDescription>Server configuration</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <label className="text-sm font-medium">Port</label>
              <input
                type="number"
                value={formData.serverPort}
                onChange={(e) => setFormData({ ...formData, serverPort: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border rounded-md bg-background"
              />
            </div>
          </CardContent>
        </Card>

        <div className="flex gap-4">
          <Button type="submit" disabled={updateMutation.isPending}>
            <Save className="mr-2 h-4 w-4" />
            {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
          </Button>
          <Button type="button" variant="outline" onClick={() => window.location.reload()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Reload Config
          </Button>
        </div>
      </form>
    </div>
  )
}
