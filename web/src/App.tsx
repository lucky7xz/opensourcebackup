import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Sidebar }      from './components/Sidebar'
import { Dashboard }    from './pages/Dashboard'
import { Systems }      from './pages/Systems'
import { Agents }       from './pages/Agents'
import { Policies }     from './pages/Policies'
import { Jobs }         from './pages/Jobs'
import { Snapshots }    from './pages/Snapshots'
import { RestoreTests } from './pages/RestoreTests'
import { Repositories } from './pages/Repositories'
import { Evidence }     from './pages/Evidence'
import { Settings }     from './pages/Settings'

export default function App() {
  return (
    <BrowserRouter>
      <div style={{ display:'flex', height:'100vh', overflow:'hidden' }}>
        <Sidebar />
        <main style={{ flex:1, overflowY:'auto', background:'var(--bg)' }}>
          <Routes>
            <Route path="/"              element={<Dashboard />} />
            <Route path="/systems"       element={<Systems />} />
            <Route path="/agents"        element={<Agents />} />
            <Route path="/policies"      element={<Policies />} />
            <Route path="/jobs"          element={<Jobs />} />
            <Route path="/snapshots"     element={<Snapshots />} />
            <Route path="/restore-tests" element={<RestoreTests />} />
            <Route path="/repositories"  element={<Repositories />} />
            <Route path="/evidence"      element={<Evidence />} />
            <Route path="/settings"      element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}
