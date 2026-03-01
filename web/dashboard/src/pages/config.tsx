import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Save, RefreshCw, Key, Bot, Server, Shield, CheckCircle2 } from 'lucide-react'
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
        <div className="flex items-center gap-3 text-white/60">
          <RefreshCw className="h-6 w-6 animate-spin" />
          <span>Loading configuration...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="relative">
        <div className="absolute -inset-1 bg-gradient-to-r from-violet-500/20 to-purple-500/20 rounded-2xl blur-xl opacity-50" />
        <div className="relative">
          <h1 className="text-4xl font-bold gradient-text-violet mb-2">Configuration</h1>
          <p className="text-lg text-white/60">Manage your AI settings and credentials</p>
        </div>
      </div>

      {showSaved && (
        <div className="glass rounded-xl p-4 flex items-center gap-3 bg-emerald-500/10 border-emerald-500/30 animate-in fade-in slide-in-from-top-2">
          <CheckCircle2 className="h-5 w-5 text-emerald-400" />
          <span className="text-emerald-200">Configuration updated successfully!</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* LLM Configuration */}
        <Card variant="gradient" className="overflow-hidden">
          <div className="p-6 border-b border-white/5">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-lg bg-gradient-to-br from-violet-500/20 to-purple-500/20 border border-violet-500/30">
                <Key className="h-5 w-5 text-violet-400" />
              </div>
              <div>
                <h2 className="text-xl font-semibold text-white">LLM Provider</h2>
                <p className="text-sm text-white/50">Configure your AI model API access</p>
              </div>
            </div>
          </div>
          <CardContent className="p-6">
            <div className="space-y-4">
              <label className="text-sm font-medium text-white/80">Kimi API Key</label>
              <div className="relative">
                <input
                  type="password"
                  value={formData.kimiApiKey}
                  onChange={(e) => setFormData({ ...formData, kimiApiKey: e.target.value })}
                  placeholder="sk-..."
                  className="w-full px-4 py-3 rounded-xl bg-slate-950/50 border border-white/10 text-white placeholder:text-white/30 focus:border-violet-500/50 focus:outline-none focus:ring-2 focus:ring-violet-500/20 transition-all"
                />
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  <Shield className="h-4 w-4 text-white/30" />
                </div>
              </div>
              <p className="text-xs text-white/40">
                Your API key is encrypted and never shared. Used to authenticate with Moonshot AI.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Channels */}
        <Card variant="gradient" className="overflow-hidden">
          <div className="p-6 border-b border-white/5">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-lg bg-gradient-to-br from-cyan-500/20 to-blue-500/20 border border-cyan-500/30">
                <Bot className="h-5 w-5 text-cyan-400" />
              </div>
              <div>
                <h2 className="text-xl font-semibold text-white">Channels</h2>
                <p className="text-sm text-white/50">Configure messaging platforms</p>
              </div>
            </div>
          </div>
          <CardContent className="p-6">
            <div className="space-y-4">
              <label className="text-sm font-medium text-white/80">Telegram Bot Token</label>
              <div className="relative">
                <input
                  type="password"
                  value={formData.telegramToken}
                  onChange={(e) => setFormData({ ...formData, telegramToken: e.target.value })}
                  placeholder="123456:ABC..."
                  className="w-full px-4 py-3 rounded-xl bg-slate-950/50 border border-white/10 text-white placeholder:text-white/30 focus:border-cyan-500/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 transition-all"
                />
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  <Shield className="h-4 w-4 text-white/30" />
                </div>
              </div>
              <p className="text-xs text-white/40">
                Get your bot token from @BotFather on Telegram. Leave empty to disable.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Server */}
        <Card variant="gradient" className="overflow-hidden">
          <div className="p-6 border-b border-white/5">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-lg bg-gradient-to-br from-emerald-500/20 to-teal-500/20 border border-emerald-500/30">
                <Server className="h-5 w-5 text-emerald-400" />
              </div>
              <div>
                <h2 className="text-xl font-semibold text-white">Server</h2>
                <p className="text-sm text-white/50">Server configuration</p>
              </div>
            </div>
          </div>
          <CardContent className="p-6">
            <div className="space-y-4">
              <label className="text-sm font-medium text-white/80">Port</label>
              <input
                type="number"
                value={formData.serverPort}
                onChange={(e) => setFormData({ ...formData, serverPort: parseInt(e.target.value) })}
                className="w-full px-4 py-3 rounded-xl bg-slate-950/50 border border-white/10 text-white focus:border-emerald-500/50 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 transition-all"
              />
              <p className="text-xs text-white/40">
                The port the server will listen on. Requires restart to take effect.
              </p>
            </div>
          </CardContent>
        </Card>

        <div className="flex flex-wrap gap-4 pt-4">
          <Button 
            type="submit" 
            variant="default" 
            className="gap-2 px-8"
            disabled={updateMutation.isPending}
          >
            <Save className="h-4 w-4" />
            {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
          </Button>
          <Button 
            type="button" 
            variant="glass" 
            className="gap-2"
            onClick={() => window.location.reload()}
          >
            <RefreshCw className="h-4 w-4" />
            Reload Config
          </Button>
        </div>
      </form>
    </div>
  )
}
