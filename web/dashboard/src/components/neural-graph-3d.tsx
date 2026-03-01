import { useRef, useState, useCallback, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import ForceGraph3D from 'react-force-graph-3d'
import { Brain, ZoomIn, ZoomOut, Maximize2, Rotate3D, Sparkles } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { clustersApi } from '@/lib/api'
import * as THREE from 'three'

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
  z?: number
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

export function NeuralGraph3D() {
  const fgRef = useRef<any>(null)
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)
  const [isRotating, setIsRotating] = useState(false)

  const { data: graphData, isLoading } = useQuery<GraphData>({
    queryKey: ['clusters-graph'],
    queryFn: clustersApi.getGraph,
  })

  // Auto-rotation effect
  useEffect(() => {
    if (!isRotating || !fgRef.current) return
    
    const interval = setInterval(() => {
      fgRef.current.cameraPosition({
        x: fgRef.current.camera().position.x,
        y: fgRef.current.camera().position.y,
        z: fgRef.current.camera().position.z
      })
      // Rotate around center
      const angle = 0.005
      const x = fgRef.current.camera().position.x
      const z = fgRef.current.camera().position.z
      fgRef.current.camera().position.x = x * Math.cos(angle) - z * Math.sin(angle)
      fgRef.current.camera().position.z = x * Math.sin(angle) + z * Math.cos(angle)
      fgRef.current.camera().lookAt(0, 0, 0)
    }, 50)

    return () => clearInterval(interval)
  }, [isRotating])

  const handleNodeClick = useCallback((node: GraphNode) => {
    setSelectedNode(node)
    // Fly to clicked node
    if (fgRef.current) {
      const distance = 150
      const distRatio = 1 + distance / Math.hypot(node.x || 0, node.y || 0, node.z || 0)
      
      fgRef.current.cameraPosition(
        { x: (node.x || 0) * distRatio, y: (node.y || 0) * distRatio, z: (node.z || 0) * distRatio },
        { x: node.x || 0, y: node.y || 0, z: node.z || 0 },
        1500
      )
    }
  }, [])

  const handleZoomIn = () => {
    if (fgRef.current) {
      const pos = fgRef.current.camera().position
      fgRef.current.cameraPosition({ x: pos.x * 0.8, y: pos.y * 0.8, z: pos.z * 0.8 }, null, 400)
    }
  }

  const handleZoomOut = () => {
    if (fgRef.current) {
      const pos = fgRef.current.camera().position
      fgRef.current.cameraPosition({ x: pos.x * 1.25, y: pos.y * 1.25, z: pos.z * 1.25 }, null, 400)
    }
  }

  const handleFitToView = () => {
    if (fgRef.current) {
      fgRef.current.zoomToFit(400, 50)
    }
  }

  // Custom 3D node geometry
  const getNodeGeometry = useCallback((node: GraphNode) => {
    const size = Math.sqrt(node.val) * 2 + 2
    
    if (node.type === 'cluster') {
      // Clusters are larger spheres with glow effect
      const geometry = new THREE.SphereGeometry(size, 32, 32)
      return geometry
    } else {
      // Memories are smaller spheres
      const geometry = new THREE.SphereGeometry(size * 0.5, 16, 16)
      return geometry
    }
  }, [])

  // Custom node material with glow
  const getNodeMaterial = useCallback((node: GraphNode) => {
    const color = new THREE.Color(node.color)
    
    if (node.type === 'cluster') {
      return new THREE.MeshLambertMaterial({
        color: color,
        emissive: color,
        emissiveIntensity: 0.2,
        transparent: true,
        opacity: 0.9
      })
    } else {
      return new THREE.MeshLambertMaterial({
        color: color,
        transparent: true,
        opacity: 0.7
      })
    }
  }, [])

  if (isLoading) {
    return (
      <Card className="h-[600px]">
        <CardContent className="flex items-center justify-center h-full">
          <div className="flex items-center gap-3 text-muted-foreground">
            <Brain className="h-5 w-5 animate-pulse" />
            <span>Loading 3D neural network...</span>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!graphData || graphData.nodes.length === 0) {
    return (
      <Card className="h-[600px]">
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
          Neural Network 3D
          <span className="text-xs text-muted-foreground font-normal">
            ({graphData.nodes.length} nodes, {graphData.links.length} connections)
          </span>
        </CardTitle>
        <div className="flex items-center gap-2">
          <Button 
            variant={isRotating ? "default" : "outline"} 
            size="sm" 
            onClick={() => setIsRotating(!isRotating)}
            title="Auto-rotate"
          >
            <Rotate3D className="h-4 w-4" />
          </Button>
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
        <ForceGraph3D
          ref={fgRef}
          graphData={graphData}
          nodeLabel={(node: GraphNode) => 
            `${node.name} (${node.type === 'cluster' ? 'Cluster' : 'Memory'})`
          }
          nodeColor={(node: GraphNode) => node.color}
          nodeVal={(node: GraphNode) => node.val}
          nodeThreeObject={(node: GraphNode) => {
            const mesh = new THREE.Mesh(getNodeGeometry(node), getNodeMaterial(node))
            
            // Add glow effect for clusters
            if (node.type === 'cluster') {
              const glowGeometry = new THREE.SphereGeometry(
                (Math.sqrt(node.val) * 2 + 2) * 1.2, 32, 32
              )
              const glowMaterial = new THREE.MeshBasicMaterial({
                color: new THREE.Color(node.color),
                transparent: true,
                opacity: 0.1,
                side: THREE.BackSide
              })
              const glow = new THREE.Mesh(glowGeometry, glowMaterial)
              mesh.add(glow)
            }
            
            return mesh
          }}
          linkColor={() => 'rgba(148, 163, 184, 0.4)'}
          linkWidth={(link: GraphLink) => Math.sqrt(link.value) * 0.5}
          linkOpacity={0.4}
          backgroundColor="rgba(15, 23, 42, 0)"
          onNodeClick={handleNodeClick}
          warmupTicks={100}
          cooldownTicks={50}
        />

        {/* Selected Node Info */}
        {selectedNode && (
          <div className="absolute bottom-4 left-4 p-4 rounded-lg bg-card/90 backdrop-blur border border-border shadow-lg max-w-xs">
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
        <div className="absolute top-4 right-4 p-3 rounded-lg bg-card/90 backdrop-blur border border-border shadow-lg">
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
          
          <div className="mt-3 pt-3 border-t border-border">
            <p className="text-[10px] text-muted-foreground">
              <Sparkles className="inline h-3 w-3 mr-1" />
              Click nodes to fly to them
            </p>
            <p className="text-[10px] text-muted-foreground mt-1">
              Drag to rotate • Scroll to zoom
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
