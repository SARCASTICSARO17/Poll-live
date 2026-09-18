import { Navigate, Outlet, Route, Routes } from 'react-router-dom'
import Login from './pages/Login.jsx'
import Signup from './pages/Signup.jsx'
import Dashboard from './pages/Dashboard.jsx'
import CreatePoll from './pages/CreatePoll.jsx'
import Poll from './pages/Poll.jsx'
import ProtectedRoute from './components/ProtectedRoute.jsx'
import Navbar from './components/Navbar.jsx'
import { useAuth } from './context/AuthContext.jsx'
import Loading from './components/Loading.jsx'

function RootRedirect() {
  const { token } = useAuth()
  return <Navigate to={token ? '/dashboard' : '/login'} replace />
}

function ProtectedLayout() {
  return (
    <div className="app-shell">
      <Navbar />
      <main className="container">
        <Outlet />
      </main>
    </div>
  )
}

function PublicLayout() {
  return (
    <div className="app-shell app-shell--public">
      <main className="container">
        <Outlet />
      </main>
    </div>
  )
}

export default function App() {
  const { loading } = useAuth()

  if (loading) {
    return (
      <div className="full-center">
        <Loading />
      </div>
    )
  }

  return (
    <Routes>
      <Route path="/" element={<RootRedirect />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<Signup />} />
      <Route element={<ProtectedRoute><ProtectedLayout /></ProtectedRoute>}>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/polls/new" element={<CreatePoll />} />
      </Route>
      <Route element={<PublicLayout />}>
        <Route path="/polls/:id" element={<Poll />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}