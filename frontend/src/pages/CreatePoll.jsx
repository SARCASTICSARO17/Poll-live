import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api.js'

const MIN_OPTIONS = 2
const MAX_OPTIONS = 6

export default function CreatePoll() {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const navigate = useNavigate()

  const updateOption = (index, value) => {
    setOptions((prev) => prev.map((opt, i) => (i === index ? value : opt)))
  }

  const addOption = () => {
    if (options.length < MAX_OPTIONS) {
      setOptions((prev) => [...prev, ''])
    }
  }

  const removeOption = (index) => {
    if (options.length > MIN_OPTIONS) {
      setOptions((prev) => prev.filter((_, i) => i !== index))
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')

    const trimmedQuestion = question.trim()
    const trimmedOptions = options.map((o) => o.trim()).filter(Boolean)

    if (!trimmedQuestion) {
      setError('Please enter a question.')
      return
    }
    if (trimmedOptions.length < MIN_OPTIONS) {
      setError(`Add at least ${MIN_OPTIONS} options.`)
      return
    }
    if (new Set(trimmedOptions.map((o) => o.toLowerCase())).size !== trimmedOptions.length) {
      setError('Options must be unique.')
      return
    }

    setSubmitting(true)
    try {
      const data = await api.post('/api/polls', {
        question: trimmedQuestion,
        options: trimmedOptions,
      })
      navigate(`/polls/${data.poll.id}`, { replace: true })
    } catch (err) {
      setError(err.message || 'Could not create the poll. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="page page-narrow">
      <div className="page-header">
        <h2>Create a poll</h2>
      </div>

      <form className="card form-stack" onSubmit={handleSubmit}>
        <label className="field">
          <span className="field-label">Question</span>
          <textarea
            rows="3"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            placeholder="What should we decide?"
            required
          />
        </label>

        <div className="field">
          <span className="field-label">Options</span>
          <div className="option-list">
            {options.map((option, index) => (
              <div className="option-row" key={index}>
                <span className="option-key">{String.fromCharCode(65 + index)}</span>
                <input
                  type="text"
                  value={option}
                  onChange={(e) => updateOption(index, e.target.value)}
                  placeholder={`Option ${index + 1}`}
                />
                <button
                  type="button"
                  className="btn-icon"
                  disabled={options.length <= MIN_OPTIONS}
                  onClick={() => removeOption(index)}
                  aria-label={`Remove option ${index + 1}`}
                  title="Remove option"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          {options.length < MAX_OPTIONS ? (
            <button type="button" className="btn btn-ghost btn-sm" onClick={addOption}>
              + Add option
            </button>
          ) : (
            <p className="hint">Maximum of {MAX_OPTIONS} options reached.</p>
          )}
        </div>

        {error && <p className="form-error">{error}</p>}

        <button type="submit" className="btn btn-primary" disabled={submitting}>
          {submitting ? 'Creating…' : 'Create Poll'}
        </button>
      </form>
    </div>
  )
}