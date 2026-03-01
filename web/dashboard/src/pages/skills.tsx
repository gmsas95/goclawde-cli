import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  Wrench, Search, Plus, Power, Trash2, RefreshCw,
  Cpu, Globe, MessageSquare, Brain, ShoppingCart, 
  Calendar, Heart, FileText, Cloud
} from 'lucide-react'
import { skillsApi } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import type { Skill } from '@/types'

const skillIcons: Record<string, typeof Wrench> = {
  browser: Globe,
  search: Search,
  telegram: MessageSquare,
  health: Heart,
  shopping: ShoppingCart,
  calendar: Calendar,
  documents: FileText,
  weather: Cloud,
  github: Cloud,
  intelligence: Brain,
  default: Cpu,
}

const skillColors: Record<string, string> = {
  browser: 'from-blue-500/20 to-cyan-500/20 border-blue-500/30',
  search: 'from-violet-500/20 to-purple-500/20 border-violet-500/30',
  health: 'from-rose-500/20 to-pink-500/20 border-rose-500/30',
  shopping: 'from-amber-500/20 to-orange-500/20 border-amber-500/30',
  calendar: 'from-emerald-500/20 to-teal-500/20 border-emerald-500/30',
  documents: 'from-slate-500/20 to-gray-500/20 border-slate-500/30',
  weather: 'from-sky-500/20 to-blue-500/20 border-sky-500/30',
  default: 'from-cyan-500/20 to-blue-500/20 border-cyan-500/30',
}

export function Skills() {
  const queryClient = useQueryClient()
  const [searchTerm, setSearchTerm] = useState('')

  const { data: skills, isLoading } = useQuery<Skill[]>({
    queryKey: ['skills'],
    queryFn: skillsApi.list,
  })

  const toggleMutation = useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      skillsApi.toggle(name, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills'] })
    },
  })

  const filteredSkills = skills?.filter(skill =>
    skill.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    skill.description.toLowerCase().includes(searchTerm.toLowerCase())
  )

  if (isLoading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <RefreshCw className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white">Skills Management</h1>
          <p className="text-white/60 mt-1">Manage your AI capabilities and tools</p>
        </div>
        <Button variant="glow">
          <Plus className="mr-2 h-4 w-4" />
          Install New Skill
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card variant="gradient">
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-cyan-500/20">
                <Wrench className="h-6 w-6 text-cyan-400" />
              </div>
              <div>
                <p className="text-sm text-white/60">Total Skills</p>
                <p className="text-2xl font-bold text-white">{skills?.length || 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card variant="gradient">
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/20">
                <Power className="h-6 w-6 text-emerald-400" />
              </div>
              <div>
                <p className="text-sm text-white/60">Active</p>
                <p className="text-2xl font-bold text-white">
                  {skills?.filter(s => s.enabled).length || 0}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card variant="gradient">
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-violet-500/20">
                <Cpu className="h-6 w-6 text-violet-400" />
              </div>
              <div>
                <p className="text-sm text-white/60">Total Tools</p>
                <p className="text-2xl font-bold text-white">
                  {skills?.reduce((acc, s) => acc + s.tools, 0) || 0}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card variant="gradient">
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-500/20">
                <Cloud className="h-6 w-6 text-amber-400" />
              </div>
              <div>
                <p className="text-sm text-white/60">External APIs</p>
                <p className="text-2xl font-bold text-white">3</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/40" />
        <input
          type="text"
          placeholder="Search skills..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full rounded-xl bg-white/5 border border-white/10 py-3 pl-12 pr-4 text-white placeholder:text-white/40 focus:outline-none focus:border-cyan-500/50"
        />
      </div>

      {/* Skills Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {filteredSkills?.map((skill) => {
          const Icon = skillIcons[skill.name] || skillIcons.default
          const colorClass = skillColors[skill.name] || skillColors.default
          
          return (
            <Card
              key={skill.name}
              className={cn(
                'group relative overflow-hidden transition-all duration-300',
                'bg-gradient-to-br',
                colorClass,
                skill.enabled ? 'opacity-100' : 'opacity-60'
              )}
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      'flex h-12 w-12 items-center justify-center rounded-xl',
                      'bg-white/10 backdrop-blur-sm'
                    )}>
                      <Icon className="h-6 w-6 text-white" />
                    </div>
                    <div>
                      <CardTitle className="text-lg capitalize">{skill.name}</CardTitle>
                      <CardDescription>{skill.description}</CardDescription>
                    </div>
                  </div>
                  
                  <button
                    onClick={() => toggleMutation.mutate({ 
                      name: skill.name, 
                      enabled: !skill.enabled 
                    })}
                    className={cn(
                      'rounded-lg p-2 transition-colors',
                      skill.enabled 
                        ? 'bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30' 
                        : 'bg-white/10 text-white/40 hover:bg-white/20'
                    )}
                  >
                    <Power className="h-4 w-4" />
                  </button>
                </div>
              </CardHeader>
              
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-white/60">{skill.tools} tools</span>
                    <span className="text-white/30">•</span>
                    <span className="text-sm text-white/60">v{skill.version}</span>
                  </div>
                  
                  <div className="flex gap-2">
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <RefreshCw className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0 text-rose-400">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {filteredSkills?.length === 0 && (
        <div className="text-center py-16">
          <Wrench className="h-16 w-16 text-white/20 mx-auto mb-4" />
          <p className="text-white/40">No skills found</p>
          <p className="text-sm text-white/30 mt-1">Try adjusting your search</p>
        </div>
      )}
    </div>
  )
}

function cn(...inputs: (string | undefined | null | false)[]) {
  return inputs.filter(Boolean).join(' ')
}
