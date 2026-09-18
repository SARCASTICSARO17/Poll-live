import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import api from '../api.js'
import useWebSocket from '../hooks/useWebSocket.js'
import { useAuth } from '../context/AuthContext.jsx'
import Loading from '../components/Loading.jsx'
import ResultBar from '../components/ResultBar.jsx'
import PollClosed from '../components/PollClosed.jsx'

function generateVoterId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

function getOrCreateVoterId() {
  let voterId = localStorage.getItem('voterId')
  if (!voterId) {
    voterId = generateVoterId()
    localStorage.setItem('voterId', voterId)
  }
  return voterId
}

export default function Poll() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { token } = useAuth()

  const [poll, setPoll] = useState(null)
  const [results, setResults] = useState([])
  const [totalVotes, setTotalVotes] = useState(0)
  const [isCreator, setIsCreator] = useState(false)
  const [hasVoted, setHasVoted] = useState(false)
  const [pollStatus, setPollStatus] = useState('open')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [votingId, setVotingId] = useState(null)
  const [copying, setCopying] = useState(false)
  const [actionError, setActionError] = useState('')

  const live = useWebSocket(id)

  const closed = pollStatus === 'closed' || live.status === 'closed'

  const maxVotes = useMemo(
    () => results.reduce((max, r) => Math.max(max, r.votes || 0), 0),
    [results]
  )

  const loadPoll = useCallback(async (opts = {}) => {
    try {
      const data = await api.get(`/api/polls/${id}`)
      setPoll(data.poll)
      setResults(data.results || [])
      setTotalVotes(data.totalVotes || 0)
      setIsCreator(Boolean(data.isCreator))
      setHasVoted(Boolean(data.hasVoted))
      setPollStatus(data.poll?.status || 'open')
      setError('')
      if (opts.showSpinner !== false) {
        setLoading(false)
      }
    } catch (err) {
      setError(err.message || 'Could not load this poll.')
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    setLoading(true)
    loadPoll()
  }, [loadPoll])

  const handleVote = async (optionId) => {
    if (votingId || hasVoted || closed) return
    setActionError('')
    setVotingId(optionId)
    try {
      const config = {}
      if (!token) {
        config.headers = { 'X-Voter-Id': getOrCreateVoterId() }
      }
      await api.post(`/api/polls/${id}/vote`, { optionId }, config)
      setHasVoted(true)
      setLoading(true)
      await loadPoll({ showSpinner: false })
      setLoading(false)
    } catch (err) {
      const message = err.message || 'Could not submit your vote.'
      if (/already voted/i.test(message)) {
        setHasVoted(true)
        setLoading(true)
        await loadPoll({ showSpinner: false }).catch(() => {})
        setLoading(false)
      } else {
        setActionError(message)
      }
    } finally {
      setVotingId(null)
    }
  }

  const handleClose = async () => {
    if (!isCreator) return
    setActionError('')
    try {
      await api.patch(`/api/polls/${id}/status`, { status: 'closed' })
      setPollStatus('closed')
    } catch (err) {
      setActionError(err.message || 'Could not close the poll.')
    }
  }

  const handleDelete = async () => {
    if (!isCreator) return
    if (!window.confirm('Delete this poll permanently?')) return
    setActionError('')
    try {
      await api.delete(`/api/polls/${id}`)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setActionError(err.message || 'Could not delete the poll.')
    }
  }

  const handleCopyLink = async () => {
    const url = window.location.href
    try {
      await navigator.clipboard.writeText(url)
      setCopying(true)
    } catch {
      const textarea = document.createElement('textarea')
      textarea.value = url
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      try {
        document.execCommand('copy')
        setCopying(true)
      } catch {
        // Clipboard unavailable.
      }
      document.body.removeChild(textarea)
    }
    setTimeout(() => setCopying(false), 2000)
  }

  const displayedResults = live.results && live.results.length > 0 ? live.results : results
  const displayedTotalVotes = live.totalVotes > 0 ? live.totalVotes : totalVotes

  const canVote = !hasVoted && !closed && !!poll

  if (loading) {
    return (
      <div className="center-slot page-pad">
        <Loading />
      </div>
    )
  }

  if (error && !poll) {
    return (
      <div className="center-slot page-pad">
        <div className="empty-state">
          <p>{error}</p>
          <button type="button" className="btn btn-ghost" onClick={() => loadPoll()}>
            Try again
          </button>
        </div>
      </div>
    )
  }

  if (!poll) {
    return null
  }

  return (
    <div className="page page-narrow">
      {closed && <PollClosed />}

      <header className="poll-hero card">
        <div className="poll-hero-top">
          <h1 className="poll-question">{poll.question}</h1>
          <span className={`status-pill ${closed ? 'status-closed' : 'status-live'}`}>
            {closed ? 'Closed' : 'Live'}
          </span>
        </div>
        <div className="poll-hero-meta">
          <span>
            {displayedTotalVotes} vote{displayedTotalVotes === 1 ? '' : 's'}
          </span>
          {isCreator && !closed && (
            <span className="live-badge">
              <span className="live-dot" />
              Live
            </span>
          )}
          <button type="button" className="btn btn-ghost btn-sm" onClick={handleCopyLink}>
            {copying ? 'Copied!' : 'Copy Link'}
          </button>
        </div>
      </header>

      {!live.connected && (
        <p className="hint">
          {pollStatus === 'open' && !closed
            ? 'Live updates unavailable — results may be stale.'
            : 'Live results are no longer available for this poll.'}
        </p>
      )}

      {canVote && (
        <section className="card vote-section">
          <h2 className="section-title">Vote</h2>
          <div className="vote-options">
            {poll.options.map((option, index) => (
              <button
                key={option.id}
                type="button"
                className="vote-option"
                disabled={votingId !== null}
                onClick={() => handleVote(option.id)}
              >
                <span className="vote-key">{String.fromCharCode(65 + index)}</span>
                <span className="vote-text">{option.text}</span>
                {votingId === option.id && <Loading size={16} />}
              </button>
            ))}
          </div>
          {hasVoted && <p className="hint">You already voted. Results update live below.</p>}
          {actionError && <p className="form-error">{actionError}</p>}
        </section>
      )}

      <section className="card results-section">
        <div className="results-header">
          <h2 className="section-title">Live results</h2>
          {!closed && !hasVoted && <span className="hint">Votes update in realtime.</span>}
        </div>
        {displayedResults.length > 0 ? (
          <div className="results-list">
            {displayedResults.map((result) => (
              <ResultBar key={result.optionId} result={result} maxVotes={maxVotes} />
            ))}
          </div>
        ) : (
          <p className="hint">No votes yet — be the first!</p>
        )}
      </section>

      {isCreator && (
        <div className="poll-actions">
          {!closed && (
            <button type="button" className="btn btn-ghost" onClick={handleClose}>
              Close Poll
            </button>
          )}
          <button type="button" className="btn btn-danger" onClick={handleDelete}>
            Delete
          </button>
          {actionError && <p className="form-error">{actionError}</p>}
        </div>
      )}
    </div>
  )
}