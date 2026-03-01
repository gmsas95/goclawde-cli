import { useRef, useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import ForceGraph2D from 'react-force-graph-2d'
import { Brain, ZoomIn, ZoomOut, Maximize2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { clustersApi } from '@/lib/api'

interface GraphNode {
  id: string
  name: string
  val: number
  color: string
  confidence?: number
  type: 'cluster' | 'memory'
  clusterId?: string
  x?: number
  y?: number
}

interface GraphLink {
  source: string
  target: string
  value: number
}

interface GraphData {
  nodes: GraphNode[]
  links: GraphLink[]
}

export function NeuralGraph() {
  const fgRef = useRef<any>(null)
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)
  const [, setHoveredNode] = useState<GraphNode | null>(null)

  const { data: graphData, isLoading } = useQuery<GraphData>({
    queryKey: ['clusters-graph'],
    queryFn: clustersApi.getGraph,
  })

  const handleNodeClick = useCallback((node: GraphNode) => {
    setSelectedNode(node)
    // Center view on clicked node
    if (fgRef.current) {
      fgRef.current.centerAt(node.x, node.y, 1000)
      fgRef.current.zoom(2, 1000)
    }
  }, [])

  const handleZoomIn = () => {
    if (fgRef.current) {
      fgRef.current.zoom(fgRef.current.zoom() * 1.3, 400)
    }
  }

  const handleZoomOut = () => {
    if (fgRef.current) {
      fgRef.current.zoom(fgRef.current.zoom() / 1.3, 400)
    }
  }

  const handleFitToView = () => {
    if (fgRef.current) {
      fgRef.current.zoomToFit(400, 50)
    }
  }

  if (isLoading) {
    return (
      <Card className="h-[500px]">
        <CardContent className="flex items-center justify-center h-full">
          <div className="flex items-center gap-3 text-muted-foreground">
            <Brain className="h-5 w-5 animate-pulse" />
            <span>Loading neural network...</span>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!graphData || graphData.nodes.length === 0) {
    return (
      <Card className="h-[500px]">
        <CardContent className="flex items-center justify-center h-full">
          <div className="text-center">
            <Brain className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">No neural clusters found</p>
            <p className="text-sm text-muted-foreground mt-2">Start using the AI to create memories</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="h-[600px]">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Brain className="h-4 w-4" />
          Neural Network Graph
          <span className="text-xs text-muted-foreground font-normal">
            ({graphData.nodes.length} nodes, {graphData.links.length} connections)
          </span>
        </CardTitle>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleZoomOut}>
            <ZoomOut className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="sm" onClick={handleZoomIn}>
            <ZoomIn className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="sm" onClick={handleFitToView}>
            <Maximize2 className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      
      <CardContent className="relative h-[calc(100%-60px)] p-0 overflow-hidden rounded-b-lg">
        <ForceGraph2D
          ref={fgRef}
          graphData={graphData}
          nodeLabel={(node: GraphNode) => 
            `${node.name} (${node.type === 'cluster' ? 'Cluster' : 'Memory'})`
          }
          nodeColor={(node: GraphNode) => node.color}
          nodeVal={(node: GraphNode) => node.val}
          linkColor={() => 'rgba(148, 163, 184, 0.3)'}
          linkWidth={(link: GraphLink) => Math.sqrt(link.value)}
          backgroundColor="transparent"
          onNodeClick={handleNodeClick}
          onNodeHover={setHoveredNode}
          warmupTicks={100}
          cooldownTicks={50}
          enableZoomInteraction={true}
          enablePanInteraction={true}
          enablePointerInteraction={true}
          nodeCanvasObject={(node: GraphNode, ctx: CanvasRenderingContext2D, globalScale: number) => {
            const label = node.name
            const fontSize = node.type === 'cluster' ? 12 : 8
            ctx.font = `${node.type === 'cluster' ? 'bold' : 'normal'} ${fontSize}px Inter, sans-serif`
            
            // Draw node circle
            const size = Math.sqrt(node.val) * 3 + 2
            ctx.beginPath()
            ctx.arc(node.x!, node.y!, size, 0, 2 * Math.PI)
            ctx.fillStyle = node.color
            ctx.fill()
            
            // Draw border for clusters
            if (node.type === 'cluster') {
              ctx.strokeStyle = 'rgba(255,255,255,0.3)'
              ctx.lineWidth = 2
              ctx.stroke()
            }
            
            // Draw label only for clusters or when zoomed in
            if (node.type === 'cluster' || globalScale > 1.5) {
              const textWidth = ctx.measureText(label).width
              const bckgDimensions = [textWidth, fontSize].map(n => n + fontSize * 0.2)
              
              ctx.fillStyle = 'rgba(15, 23, 42, 0.8)'
              ctx.fillRect(
                node.x! - bckgDimensions[0] / 2,
                node.y! + size + 4,
                bckgDimensions[0],
                bckgDimensions[1]
              )
              
              ctx.textAlign = 'center'
              ctx.textBaseline = 'middle'
              ctx.fillStyle = '#F8FAFC'
              ctx.fillText(label, node.x!, node.y! + size + 4 + fontSize / 2)
            }
          }}
          width={undefined}
          height={undefined}
        />

        {/* Selected Node Info */}
        {selectedNode && (
          <div className="absolute bottom-4 left-4 p-4 rounded-lg bg-card border border-border shadow-lg max-w-xs">
            <div className="flex items-center gap-2 mb-2">
              <div 
                className="w-3 h-3 rounded-full" 
                style={{ backgroundColor: selectedNode.color }}
              />
              <span className="font-semibold text-sm">{selectedNode.name}</span>
            </div>
            <div className="space-y-1 text-xs text-muted-foreground">
              <p>Type: <span className="text-foreground capitalize">{selectedNode.type}</span></p>
              {selectedNode.confidence && (
                <p>Confidence: <span className="text-foreground">{Math.round(selectedNode.confidence * 100)}%</span></p>
              )}
              <p>Size: <span className="text-foreground">{selectedNode.val}</span></p>
            </div>
            <Button 
              variant="ghost" 
              size="sm" 
              className="mt-2 h-7 text-xs"
              onClick={() => setSelectedNode(null)}
            >
              Close
            </Button>
          </div>
        )}

        {/* Legend */}
        <div className="absolute top-4 right-4 p-3 rounded-lg bg-card border border-border shadow-lg">
          <p className="text-xs font-medium mb-2">Legend</p>
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-xs text-muted-foreground">High confidence</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-amber-500" />
              <span className="text-xs text-muted-foreground">Medium confidence</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-red-500" />
              <span className="text-xs text-muted-foreground">Low confidence</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-indigo-500" />
              <span className="text-xs text-muted-foreground">Memory</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
