import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Save, RefreshCw, Key, Bot, Server, CheckCircle2 } from 'lucide-react'
import { configApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { useStore } from '@/stores/app'
import type { Config } from '@/types'

export function Config() {
  const queryClient = useQueryClient()
  const { setConfig } = useStore()
  const [showSaved, setShowSaved] = useState(false)
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
      setShowSaved(true)
      setTimeout(() => setShowSaved(false), 3000)
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
      <div className="flex items-center justify-center h-[60vh]">
        <div className="flex items-center gap-3 text-muted-foreground">
          <RefreshCw className="h-5 w-5 animate-spin" />
          <span>Loading configuration...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Configuration</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage your AI settings and credentials</p>
      </div>

      {showSaved && (
        <div className="flex items-center gap-2 p-4 rounded-lg bg-accent/10 text-accent">
          <CheckCircle2 className="h-4 w-4" />
          <span className="text-sm font-medium">Configuration updated successfully</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* LLM Configuration */}
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3 mb-5">
              <div className="p-2 rounded-md bg-muted">
                <Key className="h-4 w-4" />
              </div>
              <div>
                <h2 className="font-medium">LLM Provider</h2>
                <p className="text-sm text-muted-foreground">Configure your AI model API access</p>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Kimi API Key</label>
              <input
                type="password"
                value={formData.kimiApiKey}
                onChange={(e) => setFormData({ ...formData, kimiApiKey: e.target.value })}
                placeholder="sk-..."
                className="w-full px-3 py-2 rounded-md bg-background border border-border text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none transition-colors"
              />
              <p className="text-xs text-muted-foreground">Your API key is encrypted and never shared.</p>
            </div>
          </CardContent>
        </Card>

        {/* Channels */}
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3 mb-5">
              <div className="p-2 rounded-md bg-muted">
                <Bot className="h-4 w-4" />
              </div>
              <div>
                <h2 className="font-medium">Channels</h2>
                <p className="text-sm text-muted-foreground">Configure messaging platforms</p>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Telegram Bot Token</label>
              <input
                type="password"
                value={formData.telegramToken}
                onChange={(e) => setFormData({ ...formData, telegramToken: e.target.value })}
                placeholder="123456:ABC..."
                className="w-full px-3 py-2 rounded-md bg-background border border-border text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none transition-colors"
              />
              <p className="text-xs text-muted-foreground">Get your token from @BotFather on Telegram.</p>
            </div>
          </CardContent>
        </Card>

        {/* Server */}
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3 mb-5">
              <div className="p-2 rounded-md bg-muted">
                <Server className="h-4 w-4" />
              </div>
              <div>
                <h2 className="font-medium">Server</h2>
                <p className="text-sm text-muted-foreground">Server configuration</p>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Port</label>
              <input
                type="number"
                value={formData.serverPort}
                onChange={(e) => setFormData({ ...formData, serverPort: parseInt(e.target.value) })}
                className="w-full px-3 py-2 rounded-md bg-background border border-border text-foreground focus:border-primary focus:outline-none transition-colors"
              />
              <p className="text-xs text-muted-foreground">Requires restart to take effect.</p>
            </div>
          </CardContent>
        </Card>

        <div className="flex gap-3">
          <Button type="submit" disabled={updateMutation.isPending}>
            <Save className="mr-2 h-4 w-4" />
            {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
          </Button>
          <Button type="button" variant="outline" onClick={() => window.location.reload()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Reload
          </Button>
        </div>
      </form>
    </div>
  )
}
