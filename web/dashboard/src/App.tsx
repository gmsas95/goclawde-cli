import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Dashboard } from '@/pages/dashboard'
import { Config } from '@/pages/config'
import { Skills } from '@/pages/skills'
import { Memory } from '@/pages/memory'
import { Jobs } from '@/pages/jobs'
import { Persona } from '@/pages/persona'
import { Logs } from '@/pages/logs'

// Chat is still under development
const Chat = () => (
  <div className="flex h-[60vh] items-center justify-center">
    <div className="text-center">
      <div className="text-6xl mb-4">💬</div>
      <h2 className="text-2xl font-bold text-white mb-2">Chat Interface</h2>
      <p className="text-white/60">Coming soon - Web-based chat with your AI</p>
    </div>
  </div>
)

const NotFound = () => (
  <div className="flex h-[60vh] items-center justify-center">
    <div className="text-center">
      <div className="text-8xl font-bold bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent mb-4">404</div>
      <h2 className="text-2xl font-bold text-white mb-2">Page Not Found</h2>
      <p className="text-white/60 mb-6">The page you're looking for doesn't exist</p>
      <a 
        href="/" 
        className="inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-cyan-500 to-blue-600 px-6 py-3 text-white font-medium hover:from-cyan-400 hover:to-blue-500 transition-all"
      >
        Go Home
      </a>
    </div>
  </div>
)

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/chat" element={<Chat />} />
            <Route path="/skills" element={<Skills />} />
            <Route path="/memory" element={<Memory />} />
            <Route path="/jobs" element={<Jobs />} />
            <Route path="/persona" element={<Persona />} />
            <Route path="/config" element={<Config />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
