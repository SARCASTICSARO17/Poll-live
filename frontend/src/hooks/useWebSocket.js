import { useCallback, useEffect, useRef, useState } from 'react'

const WS_BASE = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'
const MAX_RETRY_DELAY = 30000
const HEARTBEAT_INTERVAL = 25000

function buildWsUrl(pollId) {
  const base = WS_BASE.replace(/\/+$/, '')
  return `${base}/api/polls/${pollId}/ws`
}

export default function useWebSocket(pollId) {
  const [results, setResults] = useState([])
  const [totalVotes, setTotalVotes] = useState(0)
  const [status, setStatus] = useState(null)
  const [connected, setConnected] = useState(false)

  const socketRef = useRef(null)
  const retryRef = useRef(0)
  const shouldReconnectRef = useRef(true)
  const heartbeatRef = useRef(null)
  const pollIdRef = useRef(pollId)

  useEffect(() => {
    pollIdRef.current = pollId
  }, [pollId])

  const clearHeartbeat = useCallback(() => {
    if (heartbeatRef.current) {
      clearInterval(heartbeatRef.current)
      heartbeatRef.current = null
    }
  }, [])

  const startHeartbeat = useCallback(() => {
    clearHeartbeat()
    heartbeatRef.current = setInterval(() => {
      const socket = socketRef.current
      if (socket && socket.readyState === WebSocket.OPEN) {
        try {
          socket.send(JSON.stringify({ type: 'ping' }))
        } catch {
          // Heartbeat is best-effort; browsers can't send native WS pings.
        }
      }
    }, HEARTBEAT_INTERVAL)
  }, [clearHeartbeat])

  const connect = useCallback(() => {
    const pollIdNow = pollIdRef.current
    if (!pollIdNow) return

    const current = socketRef.current
    if (
      current &&
      (current.readyState === WebSocket.OPEN || current.readyState === WebSocket.CONNECTING)
    ) {
      return
    }

    clearHeartbeat()
    const socket = new WebSocket(buildWsUrl(pollIdNow))
    socketRef.current = socket

    socket.onopen = () => {
      retryRef.current = 0
      setConnected(true)
      startHeartbeat()
    }

    socket.onmessage = (event) => {
      let msg
      try {
        msg = JSON.parse(event.data)
      } catch {
        return
      }
      if (!msg || typeof msg !== 'object') return

      if (msg.type === 'poll_update') {
        if (Array.isArray(msg.results)) {
          setResults(msg.results)
        }
        if (msg.totalVotes != null) {
          setTotalVotes(msg.totalVotes)
        }
      } else if (msg.type === 'poll_status' && msg.status) {
        setStatus(msg.status)
      }
    }

    socket.onerror = () => {
      // onclose handles reconnection.
    }

    socket.onclose = () => {
      setConnected(false)
      clearHeartbeat()
      if (!shouldReconnectRef.current) {
        socketRef.current = null
        return
      }
      const delay = Math.min(1000 * 2 ** retryRef.current, MAX_RETRY_DELAY)
      retryRef.current += 1
      setTimeout(() => {
        if (shouldReconnectRef.current) {
          connect()
        }
      }, delay)
    }
  }, [clearHeartbeat, startHeartbeat])

  useEffect(() => {
    shouldReconnectRef.current = true
    retryRef.current = 0
    setConnected(false)
    connect()

    return () => {
      shouldReconnectRef.current = false
      clearHeartbeat()
      const current = socketRef.current
      if (current) {
        current.onclose = null
        try {
          current.close()
        } catch {
          // Ignore
        }
        socketRef.current = null
      }
    }
  }, [pollId, connect, clearHeartbeat])

  return { results, totalVotes, status, connected }
}