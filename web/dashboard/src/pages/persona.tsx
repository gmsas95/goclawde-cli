import { useState } from 'react'
import { Sparkles, MessageSquare, Bot, Save, RotateCcw, Smile, Zap, Shield, Palette, Volume2 } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface PersonalityTrait {
  id: string
  name: string
  description: string
  value: number
  icon: typeof Smile
  color: string
}

const defaultTraits: PersonalityTrait[] = [
  { id: 'friendly', name: 'Friendly', description: 'Warm and approachable demeanor', value: 75, icon: Smile, color: 'from-pink-500/20 to-rose-500/20 text-pink-400' },
  { id: 'creative', name: 'Creative', description: 'Imaginative problem solving', value: 80, icon: Sparkles, color: 'from-violet-500/20 to-purple-500/20 text-violet-400' },
  { id: 'analytical', name: 'Analytical', description: 'Data-driven decision making', value: 70, icon: Bot, color: 'from-cyan-500/20 to-blue-500/20 text-cyan-400' },
  { id: 'assertive', name: 'Assertive', description: 'Direct and confident communication', value: 60, icon: Zap, color: 'from-amber-500/20 to-orange-500/20 text-amber-400' },
  { id: 'formal', name: 'Formal', description: 'Professional tone and style', value: 45, icon: Shield, color: 'from-slate-500/20 to-slate-400/20 text-slate-400' },
  { id: 'verbose', name: 'Verbose', description: 'Detailed and thorough responses', value: 65, icon: MessageSquare, color: 'from-emerald-500/20 to-teal-500/20 text-emerald-400' },
]

const voiceStyles = [
  { id: 'casual', name: 'Casual', description: 'Relaxed and conversational' },
  { id: 'professional', name: 'Professional', description: 'Business-appropriate formal' },
  { id: 'technical', name: 'Technical', description: 'Precise and jargon-friendly' },
  { id: 'friendly', name: 'Friendly', description: 'Warm and personal' },
  { id: 'witty', name: 'Witty', description: 'Humorous and playful' },
]

export function Persona() {
  const [traits, setTraits] = useState<PersonalityTrait[]>(defaultTraits)
  const [systemPrompt, setSystemPrompt] = useState(`You are a helpful AI assistant with a warm, friendly personality. You communicate clearly and efficiently while maintaining a conversational tone. You are knowledgeable about a wide range of topics and enjoy helping users with their questions and tasks.`)
  const [selectedVoice, setSelectedVoice] = useState('friendly')
  const [greeting, setGreeting] = useState('Hey there! How can I help you today?')
  const [hasChanges, setHasChanges] = useState(false)

  const updateTrait = (id: string, value: number) => {
    setTraits(traits.map(t => t.id === id ? { ...t, value } : t))
    setHasChanges(true)
  }

  const handleSave = () => {
    setHasChanges(false)
    // TODO: Save to API
  }

  const handleReset = () => {
    setTraits(defaultTraits)
    setSystemPrompt(`You are a helpful AI assistant with a warm, friendly personality. You communicate clearly and efficiently while maintaining a conversational tone.`)
    setSelectedVoice('friendly')
    setGreeting('Hey there! How can I help you today?')
    setHasChanges(false)
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="relative">
        <div className="absolute -inset-1 bg-gradient-to-r from-pink-500/20 to-rose-500/20 rounded-2xl blur-xl opacity-50" />
        <div className="relative flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-bold gradient-text-pink mb-2">Persona Editor</h1>
            <p className="text-lg text-white/60">Customize your AI's personality and behavior</p>
          </div>
          {hasChanges && (
            <div className="flex gap-2">
              <Button variant="glass" className="gap-2" onClick={handleReset}>
                <RotateCcw className="h-4 w-4" />
                Reset
              </Button>
              <Button variant="default" className="gap-2" onClick={handleSave}>
                <Save className="h-4 w-4" />
                Save Changes
              </Button>
            </div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Personality Traits */}
        <div className="lg:col-span-2 space-y-6">
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-6 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-lg bg-gradient-to-br from-pink-500/20 to-rose-500/20 border border-pink-500/30">
                  <Sparkles className="h-5 w-5 text-pink-400" />
                </div>
                <div>
                  <h2 className="text-xl font-semibold text-white">Personality Traits</h2>
                  <p className="text-sm text-white/50">Adjust sliders to shape personality</p>
                </div>
              </div>
            </div>
            <CardContent className="p-6">
              <div className="space-y-6">
                {traits.map((trait) => {
                  const Icon = trait.icon
                  return (
                    <div key={trait.id} className="space-y-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className={`p-2 rounded-lg bg-gradient-to-br ${trait.color} border`}>
                            <Icon className="h-4 w-4" />
                          </div>
                          <div>
                            <p className="font-medium text-white">{trait.name}</p>
                            <p className="text-xs text-white/40">{trait.description}</p>
                          </div>
                        </div>
                        <span className="text-lg font-bold text-white">{trait.value}%</span>
                      </div>
                      <input
                        type="range"
                        min="0"
                        max="100"
                        value={trait.value}
                        onChange={(e) => updateTrait(trait.id, parseInt(e.target.value))}
                        className="w-full h-2 rounded-lg bg-slate-950/50 appearance-none cursor-pointer accent-pink-500"
                        style={{
                          background: `linear-gradient(to right, rgba(236, 72, 153, 0.5) ${trait.value}%, rgba(15, 23, 42, 0.5) ${trait.value}%)`
                        }}
                      />
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>

          {/* System Prompt */}
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-6 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-lg bg-gradient-to-br from-violet-500/20 to-purple-500/20 border border-violet-500/30">
                  <Bot className="h-5 w-5 text-violet-400" />
                </div>
                <div>
                  <h2 className="text-xl font-semibold text-white">System Prompt</h2>
                  <p className="text-sm text-white/50">Define core behavior and context</p>
                </div>
              </div>
            </div>
            <CardContent className="p-6">
              <textarea
                value={systemPrompt}
                onChange={(e) => {
                  setSystemPrompt(e.target.value)
                  setHasChanges(true)
                }}
                rows={6}
                className="w-full px-4 py-3 rounded-xl bg-slate-950/50 border border-white/10 text-white placeholder:text-white/30 focus:border-violet-500/50 focus:outline-none resize-none"
                placeholder="Enter system prompt..."
              />
              <p className="mt-2 text-xs text-white/40">This defines the core personality and instructions for your AI assistant.</p>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Voice Style */}
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-4 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gradient-to-br from-amber-500/20 to-orange-500/20 border border-amber-500/30">
                  <Volume2 className="h-4 w-4 text-amber-400" />
                </div>
                <h3 className="font-semibold text-white">Voice Style</h3>
              </div>
            </div>
            <CardContent className="p-4">
              <div className="space-y-2">
                {voiceStyles.map((style) => (
                  <button
                    key={style.id}
                    onClick={() => {
                      setSelectedVoice(style.id)
                      setHasChanges(true)
                    }}
                    className={`w-full p-3 rounded-lg text-left transition-all ${
                      selectedVoice === style.id
                        ? 'bg-pink-500/20 border border-pink-500/30'
                        : 'hover:bg-white/5 border border-transparent'
                    }`}
                  >
                    <p className="font-medium text-white text-sm">{style.name}</p>
                    <p className="text-xs text-white/40">{style.description}</p>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Greeting */}
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-4 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gradient-to-br from-emerald-500/20 to-teal-500/20 border border-emerald-500/30">
                  <MessageSquare className="h-4 w-4 text-emerald-400" />
                </div>
                <h3 className="font-semibold text-white">Greeting Message</h3>
              </div>
            </div>
            <CardContent className="p-4">
              <input
                type="text"
                value={greeting}
                onChange={(e) => {
                  setGreeting(e.target.value)
                  setHasChanges(true)
                }}
                className="w-full px-3 py-2 rounded-lg bg-slate-950/50 border border-white/10 text-white text-sm focus:border-emerald-500/50 focus:outline-none"
                placeholder="Enter greeting..."
              />
              <p className="mt-2 text-xs text-white/40">Shown when starting a new conversation.</p>
            </CardContent>
          </Card>

          {/* Preview */}
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-4 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/20 to-blue-500/20 border border-cyan-500/30">
                  <Palette className="h-4 w-4 text-cyan-400" />
                </div>
                <h3 className="font-semibold text-white">Preview</h3>
              </div>
            </div>
            <CardContent className="p-4">
              <div className="p-3 rounded-lg bg-slate-950/50 border border-white/5">
                <p className="text-sm text-white/80 mb-2">AI Assistant</p>
                <p className="text-sm text-white">{greeting}</p>
              </div>
              <div className="mt-3 p-3 rounded-lg bg-pink-500/10 border border-pink-500/20">
                <p className="text-sm text-pink-200">Personality: {traits.filter(t => t.value > 50).map(t => t.name).join(', ') || 'Balanced'}</p>
                <p className="text-sm text-pink-200/70 mt-1">Voice: {voiceStyles.find(v => v.id === selectedVoice)?.name}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
