import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Dashboard } from '@/pages/dashboard'
import { Config } from '@/pages/config'

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
            <Route path="/config" element={<Config />} />
            <Route path="*" element={
              <div className="text-center">
                <h2 className="text-2xl font-bold">404</h2>
                <p className="text-muted-foreground">Page not found</p>
              </div>
            } />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
