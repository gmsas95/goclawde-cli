import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Brain, Database, Network, Zap, Search, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { clustersApi } from '@/lib/api'

interface Cluster {
  id: string
  name: string
  type: string
  size: number
  last_accessed: string
  importance: number
  connections: number
  confidence: number
}

export function Memory() {
  const [searchQuery, setSearchQuery] = useState('')

  const { data: clusters = [], isLoading } = useQuery<Cluster[]>({
    queryKey: ['clusters'],
    queryFn: clustersApi.list,
  })

  const filteredClusters = clusters.filter(cluster =>
    cluster.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const totalSize = clusters.reduce((acc, c) => acc + c.size, 0)
  const totalConnections = clusters.reduce((acc, c) => acc + c.connections, 0)

  if (isLoading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex items-center gap-3 text-muted-foreground">
          <Brain className="h-5 w-5 animate-pulse" />
          <span>Loading memory clusters...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Neural Memory</h1>
          <p className="text-sm text-muted-foreground mt-1">Explore memory clusters and context</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Clusters</CardTitle>
            <Brain className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{clusters.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Memory Entries</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalSize}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Connections</CardTitle>
            <Network className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalConnections}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Avg Confidence</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {clusters.length > 0 
                ? Math.round(clusters.reduce((acc, c) => acc + c.confidence, 0) / clusters.length * 100)
                : 0}%
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Search memory clusters..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full pl-10 pr-4 py-2 rounded-lg bg-card border border-border text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none transition-colors"
        />
      </div>

      {/* Clusters Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {filteredClusters.map((cluster) => (
          <Card key={cluster.id} className="cursor-pointer hover:border-primary/50 transition-colors">
            <CardContent className="p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-md bg-muted">
                  <Brain className="h-4 w-4" />
                </div>
                <h3 className="font-medium truncate">{cluster.name}</h3>
              </div>

              <div className="grid grid-cols-3 gap-3 text-center">
                <div className="p-2 rounded-md bg-muted">
                  <p className="text-lg font-semibold">{cluster.size}</p>
                  <p className="text-xs text-muted-foreground">Entries</p>
                </div>
                <div className="p-2 rounded-md bg-muted">
                  <p className="text-lg font-semibold">{cluster.connections}</p>
                  <p className="text-xs text-muted-foreground">Links</p>
                </div>
                <div className="p-2 rounded-md bg-muted">
                  <p className="text-lg font-semibold">{cluster.importance}%</p>
                  <p className="text-xs text-muted-foreground">Weight</p>
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Clock className="h-3 w-3" />
                  <span>{cluster.last_accessed}</span>
                </div>
                <span className="px-2 py-0.5 rounded-full bg-muted capitalize">{cluster.type}</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredClusters.length === 0 && (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Brain className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">No memory clusters found</p>
          </div>
        </div>
      )}
    </div>
  )
}
