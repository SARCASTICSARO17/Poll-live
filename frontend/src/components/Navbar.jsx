import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export default function Navbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <header className="navbar">
      <div className="container navbar-inner">
        <Link to="/dashboard" className="brand">
          PollLive
        </Link>
        <nav className="nav-links">
          <Link to="/dashboard">Dashboard</Link>
          <Link to="/polls/new">New Poll</Link>
          <span className="nav-user">{user ? user.email : ''}</span>
          <button type="button" className="btn btn-ghost" onClick={handleLogout}>
            Logout
          </button>
        </nav>
      </div>
    </header>
  )
}