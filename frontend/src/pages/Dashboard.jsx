import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api.js'
import Loading from '../components/Loading.jsx'

function PollCard({ entry }) {
  const poll = entry.poll
  const totalVotes = entry.totalVotes ?? 0

  return (
    <Link to={`/polls/${poll.id}`} className="poll-card">
      <div className="poll-card-top">
        <span className="poll-card-question">{poll.question}</span>
        <span className={`status-pill ${poll.status === 'closed' ? 'status-closed' : 'status-live'}`}>
          {poll.status === 'closed' ? 'Closed' : 'Live'}
        </span>
      </div>
      <div className="poll-card-meta">
        <span>{poll.options?.length ?? 0} options</span>
        <span className="poll-card-votes">{totalVotes} vote{totalVotes === 1 ? '' : 's'}</span>
      </div>
    </Link>
  )
}

export default function Dashboard() {
  const [polls, setPolls] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadPolls = useCallback(async () => {
    try {
      const data = await api.get('/api/polls/my')
      setPolls(data.polls || [])
      setError('')
    } catch (err) {
      setError(err.message || 'Could not load your polls.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadPolls()
  }, [loadPolls])

  return (
    <div className="page">
      <div className="page-header">
        <h2>Your polls</h2>
        <Link to="/polls/new" className="btn btn-primary">
          + New Poll
        </Link>
      </div>

      {loading ? (
        <div className="center-slot">
          <Loading />
        </div>
      ) : error ? (
        <div className="empty-state">
          <p>{error}</p>
          <button type="button" className="btn btn-ghost" onClick={loadPolls}>
            Try again
          </button>
        </div>
      ) : polls.length === 0 ? (
        <div className="empty-state">
          <p>You haven&apos;t created any polls yet.</p>
          <Link to="/polls/new" className="btn btn-primary">
            Create your first poll
          </Link>
        </div>
      ) : (
        <div className="poll-grid">
          {polls.map((entry) => (
            <PollCard key={entry.poll.id} entry={entry} />
          ))}
        </div>
      )}
    </div>
  )
}