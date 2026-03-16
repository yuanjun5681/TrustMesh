import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MainLayout } from '@/components/layout/MainLayout'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { ProjectListPage } from '@/pages/ProjectListPage'
import { ProjectBoardPage } from '@/pages/ProjectBoardPage'
import { ConversationPage } from '@/pages/ConversationPage'
import { AgentListPage } from '@/pages/AgentListPage'
import { useAuthStore } from '@/stores/authStore'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 10_000,
      refetchOnWindowFocus: true,
    },
  },
})

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (!isAuthenticated()) return <Navigate to="/login" replace />
  return <>{children}</>
}

function GuestRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (isAuthenticated()) return <Navigate to="/projects" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Guest Routes */}
          <Route path="/login" element={<GuestRoute><LoginPage /></GuestRoute>} />
          <Route path="/register" element={<GuestRoute><RegisterPage /></GuestRoute>} />

          {/* Protected Routes */}
          <Route
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route path="/projects" element={<ProjectListPage />} />
            <Route path="/projects/:projectId" element={<ProjectBoardPage />} />
            <Route path="/projects/:projectId/chat" element={<ConversationPage />} />
            <Route path="/agents" element={<AgentListPage />} />
          </Route>

          {/* Redirect */}
          <Route path="*" element={<Navigate to="/projects" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
