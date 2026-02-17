import React, { useEffect, useState } from 'react'
import ReactDOM from 'react-dom/client'

type Deployment = { id: number; name: string; version: string; status: string; createdAt?: string }

function App() {
  const [items, setItems] = useState<Deployment[]>([])
  const apiBase = import.meta.env.VITE_API_BASE || 'http://localhost:8080'
  const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8082/ws'

  useEffect(() => {
    fetch(`${apiBase}/api/deployments`).then(r => r.json()).then(setItems).catch(console.error)
    const ws = new WebSocket(wsUrl)
    ws.onmessage = ev => {
      const update = JSON.parse(ev.data) as Deployment
      setItems(prev => [update, ...prev.filter(p => p.id !== update.id)])
    }
    return () => ws.close()
  }, [])

  return (
    <main style={{ fontFamily: 'system-ui', padding: 24 }}>
      <h1>Deployment Event Platform</h1>
      <table border={1} cellPadding={8} style={{ borderCollapse: 'collapse', width: '100%' }}>
        <thead>
          <tr><th>ID</th><th>Name</th><th>Version</th><th>Status</th></tr>
        </thead>
        <tbody>
          {items.map(d => (
            <tr key={d.id}><td>{d.id}</td><td>{d.name}</td><td>{d.version}</td><td>{d.status}</td></tr>
          ))}
        </tbody>
      </table>
    </main>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
