import { useState } from 'react'
import { Sparkles, MessageSquare, Save, RotateCcw, Smile, Palette, Volume2 } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

const voiceStyles = [
  { id: 'casual', name: 'Casual', description: 'Relaxed and conversational' },
  { id: 'professional', name: 'Professional', description: 'Business-appropriate formal' },
  { id: 'friendly', name: 'Friendly', description: 'Warm and personal' },
  { id: 'witty', name: 'Witty', description: 'Humorous and playful' },
]

export function Persona() {
  const [systemPrompt, setSystemPrompt] = useState(`You are a helpful AI assistant with a warm, friendly personality. You communicate clearly and efficiently while maintaining a conversational tone.`)
  const [selectedVoice, setSelectedVoice] = useState('friendly')
  const [greeting, setGreeting] = useState('Hey there! How can I help you today?')
  const [hasChanges, setHasChanges] = useState(false)

  const handleSave = () => {
    setHasChanges(false)
  }

  const handleReset = () => {
    setSystemPrompt(`You are a helpful AI assistant with a warm, friendly personality.`)
    setSelectedVoice('friendly')
    setGreeting('Hey there! How can I help you today?')
    setHasChanges(false)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Persona Editor</h1>
          <p className="text-sm text-muted-foreground mt-1">Customize your AI's personality</p>
        </div>
        {hasChanges && (
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleReset}>
              <RotateCcw className="mr-2 h-4 w-4" />
              Reset
            </Button>
            <Button onClick={handleSave}>
              <Save className="mr-2 h-4 w-4" />
              Save
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* System Prompt */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <Sparkles className="h-4 w-4" />
                </div>
                <div>
                  <h2 className="font-medium">System Prompt</h2>
                  <p className="text-sm text-muted-foreground">Define core behavior and context</p>
                </div>
              </div>

              <textarea
                value={systemPrompt}
                onChange={(e) => {
                  setSystemPrompt(e.target.value)
                  setHasChanges(true)
                }}
                rows={6}
                className="w-full px-3 py-2 rounded-md bg-background border border-border text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none resize-none"
                placeholder="Enter system prompt..."
              />
            </CardContent>
          </Card>

          {/* Traits */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <Smile className="h-4 w-4" />
                </div>
                <div>
                  <h2 className="font-medium">Personality</h2>
                  <p className="text-sm text-muted-foreground">Adjust personality traits</p>
                </div>
              </div>

              <div className="space-y-4">
                {[
                  { name: 'Friendly', value: 75 },
                  { name: 'Creative', value: 80 },
                  { name: 'Analytical', value: 70 },
                  { name: 'Assertive', value: 60 },
                ].map((trait) => (
                  <div key={trait.name} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">{trait.name}</span>
                      <span className="text-sm text-muted-foreground">{trait.value}%</span>
                    </div>
                    <input
                      type="range"
                      min="0"
                      max="100"
                      defaultValue={trait.value}
                      className="w-full h-2 rounded-full bg-muted appearance-none cursor-pointer accent-primary"
                      onChange={() => setHasChanges(true)}
                    />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Voice Style */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <Volume2 className="h-4 w-4" />
                </div>
                <h3 className="font-medium">Voice Style</h3>
              </div>

              <div className="space-y-2">
                {voiceStyles.map((style) => (
                  <button
                    key={style.id}
                    onClick={() => {
                      setSelectedVoice(style.id)
                      setHasChanges(true)
                    }}
                    className={`w-full p-3 rounded-md text-left transition-colors ${
                      selectedVoice === style.id
                        ? 'bg-primary/10 border border-primary/30'
                        : 'hover:bg-muted border border-transparent'
                    }`}
                  >
                    <p className="font-medium text-sm">{style.name}</p>
                    <p className="text-xs text-muted-foreground">{style.description}</p>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Greeting */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <MessageSquare className="h-4 w-4" />
                </div>
                <h3 className="font-medium">Greeting</h3>
              </div>

              <input
                type="text"
                value={greeting}
                onChange={(e) => {
                  setGreeting(e.target.value)
                  setHasChanges(true)
                }}
                className="w-full px-3 py-2 rounded-md bg-background border border-border text-foreground text-sm focus:border-primary focus:outline-none"
                placeholder="Enter greeting..."
              />
            </CardContent>
          </Card>

          {/* Preview */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <Palette className="h-4 w-4" />
                </div>
                <h3 className="font-medium">Preview</h3>
              </div>

              <div className="p-3 rounded-md bg-muted">
                <p className="text-sm text-muted-foreground mb-2">AI Assistant</p>
                <p className="text-sm">{greeting}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
