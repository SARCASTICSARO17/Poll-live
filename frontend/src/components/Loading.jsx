export default function Loading({ size = 32 }) {
  return (
    <div className="spinner-wrap" role="status" aria-live="polite">
      <span className="spinner" style={{ width: size, height: size }} />
      <span className="sr-only">Loading…</span>
    </div>
  )
}