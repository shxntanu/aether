import { useEffect, useState } from 'react'

import './App.css'

type HealthState = 'checking' | 'ready' | 'unavailable'

function App() {
  const [health, setHealth] = useState<HealthState>('checking')

  useEffect(() => {
    const controller = new AbortController()

    async function checkHealth() {
      try {
        const response = await fetch('/api/v1/health', {
          headers: { Accept: 'application/json' },
          signal: controller.signal,
        })
        const body = (await response.json()) as { status?: string }

        setHealth(response.ok && body.status === 'ok' ? 'ready' : 'unavailable')
      } catch {
        if (!controller.signal.aborted) {
          setHealth('unavailable')
        }
      }
    }

    void checkHealth()
    return () => controller.abort()
  }, [])

  const statusMessage = {
    checking: 'Checking Aether API…',
    ready: 'Aether API is ready',
    unavailable: 'Aether API is unavailable',
  }[health]

  return (
    <main className="shell">
      <section className="welcome" aria-labelledby="page-title">
        <p className="eyebrow">Family document vault</p>
        <h1 id="page-title">Aether</h1>
        <p className="summary">
          Store household documents safely and make them easy to find again.
        </p>
        <p
          aria-label="API status"
          className={`api-status api-status--${health}`}
          role="status"
        >
          {statusMessage}
        </p>
      </section>
    </main>
  )
}

export default App
