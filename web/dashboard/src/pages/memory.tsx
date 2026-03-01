import { useState } from 'react'
import { Brain, Database, Network, Zap, Search, Trash2, Archive, Clock } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface MemoryCluster {
  id: string
  name: string
  type: 'conversation' | 'knowledge' | 'task' | 'personality'
  size: number
  lastAccessed: string
  importance: number
  connections: number
}

const mockClusters: MemoryCluster[] = [
  { id: '1', name: 'User Preferences', type: 'personality', size: 145, lastAccessed: '2 min ago', importance: 95, connections: 12 },
  { id: '2', name: 'Project Context', type: 'knowledge', size: 892, lastAccessed: '5 min ago', importance: 88, connections: 24 },
  { id: '3', name: 'Recent Conversations', type: 'conversation', size: 2341, lastAccessed: 'Just now', importance: 72, connections: 8 },
  { id: '4', name: 'Active Tasks', type: 'task', size: 67, lastAccessed: '1 hour ago', importance: 65, connections: 15 },
  { id: '5', name: 'Technical Knowledge', type: 'knowledge', size: 1567, lastAccessed: '3 hours ago', importance: 90, connections: 42 },
  { id: '6', name: 'Personal Memories', type: 'personality', size: 423, lastAccessed: '1 day ago', importance: 55, connections: 6 },
]

const getTypeColor = (type: MemoryCluster['type']) => {
  const colors = {
    conversation: 'from-amber-500/20 to-orange-500/20 border-amber-500/30 text-amber-400',
    knowledge: 'from-cyan-500/20 to-blue-500/20 border-cyan-500/30 text-cyan-400',
    task: 'from-emerald-500/20 to-teal-500/20 border-emerald-500/30 text-emerald-400',
    personality: 'from-pink-500/20 to-rose-500/20 border-pink-500/30 text-pink-400',
  }
  return colors[type]
}

const getTypeIcon = (type: MemoryCluster['type']) => {
  const icons = {
    conversation: Brain,
    knowledge: Database,
    task: Zap,
    personality: Network,
  }
  return icons[type]
}

export function Memory() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedType, setSelectedType] = useState<string | null>(null)

  const filteredClusters = mockClusters.filter(cluster => {
    const matchesSearch = cluster.name.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesType = selectedType ? cluster.type === selectedType : true
    return matchesSearch && matchesType
  })

  const totalSize = mockClusters.reduce((acc, c) => acc + c.size, 0)
  const totalConnections = mockClusters.reduce((acc, c) => acc + c.connections, 0)
  const avgImportance = Math.round(mockClusters.reduce((acc, c) => acc + c.importance, 0) / mockClusters.length)

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="relative">
        <div className="absolute -inset-1 bg-gradient-to-r from-violet-500/20 to-purple-500/20 rounded-2xl blur-xl opacity-50" />
        <div className="relative">
          <h1 className="text-4xl font-bold gradient-text-violet mb-2">Neural Memory</h1>
          <p className="text-lg text-white/60">Explore and manage your AI's memory clusters</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-violet-500/20 to-purple-500/20 border border-violet-500/30">
            <Brain className="h-6 w-6 text-violet-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Total Clusters</p>
            <p className="text-2xl font-bold text-white">{mockClusters.length}</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-cyan-500/20 to-blue-500/20 border border-cyan-500/30">
            <Database className="h-6 w-6 text-cyan-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Memory Used</p>
            <p className="text-2xl font-bold text-white">{(totalSize / 1024).toFixed(1)} MB</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-amber-500/20 to-orange-500/20 border border-amber-500/30">
            <Network className="h-6 w-6 text-amber-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Connections</p>
            <p className="text-2xl font-bold text-white">{totalConnections}</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-emerald-500/20 to-teal-500/20 border border-emerald-500/30">
            <Zap className="h-6 w-6 text-emerald-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Avg Importance</p>
            <p className="text-2xl font-bold text-white">{avgImportance}%</p>
          </div>
        </Card>
      </div>

      {/* Controls */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/40" />
          <input
            type="text"
            placeholder="Search memory clusters..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 rounded-xl bg-slate-950/50 border border-white/10 text-white placeholder:text-white/30 focus:border-violet-500/50 focus:outline-none"
          />
        </div>
        <div className="flex gap-2">
          {(['all', 'conversation', 'knowledge', 'task', 'personality'] as const).map((type) => (
            <button
              key={type}
              onClick={() => setSelectedType(type === 'all' ? null : type)}
              className={`px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                (type === 'all' && !selectedType) || selectedType === type
                  ? 'bg-violet-500/30 text-violet-300 border border-violet-500/50'
                  : 'text-white/60 hover:text-white hover:bg-white/5'
              }`}
            >
              {type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Memory Clusters Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredClusters.map((cluster) => {
          const Icon = getTypeIcon(cluster.type)
          return (
            <Card key={cluster.id} variant="gradient" className="group hover:border-violet-500/30 transition-all cursor-pointer">
              <CardContent className="p-5">
                <div className="flex items-start justify-between mb-4">
                  <div className={`p-3 rounded-lg bg-gradient-to-br ${getTypeColor(cluster.type)} border`}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Archive className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity text-red-400 hover:text-red-300">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <h3 className="text-lg font-semibold text-white mb-1">{cluster.name}</h3>
                <p className="text-sm text-white/50 mb-4 capitalize">{cluster.type}</p>

                <div className="grid grid-cols-3 gap-3 text-center">
                  <div className="p-2 rounded-lg bg-slate-950/50">
                    <p className="text-lg font-semibold text-white">{cluster.size}</p>
                    <p className="text-xs text-white/40">Entries</p>
                  </div>
                  <div className="p-2 rounded-lg bg-slate-950/50">
                    <p className="text-lg font-semibold text-cyan-400">{cluster.connections}</p>
                    <p className="text-xs text-white/40">Links</p>
                  </div>
                  <div className="p-2 rounded-lg bg-slate-950/50">
                    <p className="text-lg font-semibold text-emerald-400">{cluster.importance}%</p>
                    <p className="text-xs text-white/40">Weight</p>
                  </div>
                </div>

                <div className="mt-4 flex items-center gap-2 text-xs text-white/40">
                  <Clock className="h-3 w-3" />
                  <span>Last accessed: {cluster.lastAccessed}</span>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {filteredClusters.length === 0 && (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Brain className="h-12 w-12 text-white/20 mx-auto mb-4" />
            <p className="text-white/40">No memory clusters found</p>
          </div>
        </div>
      )}
    </div>
  )
}
