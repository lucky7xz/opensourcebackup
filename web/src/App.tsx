import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth }  from './auth'
import { Sidebar }      from './components/Sidebar'
import { Dashboard }    from './pages/Dashboard'
import { Login }        from './pages/Login'
import { Systems }      from './pages/Systems'
import { Agents }       from './pages/Agents'
import { Policies }     from './pages/Policies'
import { Jobs }         from './pages/Jobs'
import { Snapshots }    from './pages/Snapshots'
import { RestoreTests } from './pages/RestoreTests'
import { Repositories } from './pages/Repositories'
import { Evidence }     from './pages/Evidence'
import { Alerts }       from './pages/Alerts'
import { Settings }     from './pages/Settings'
import { Cockpit }     from './pages/Cockpit'

function AppShell() {
  const auth = useAuth()

  // Still checking auth status
  if (auth.loading) return (
    <div style={{ display:'flex', alignItems:'center', justifyContent:'center', height:'100vh', background:'var(--bg)', color:'var(--text-muted)', fontSize:13 }}>
      Loading…
    </div>
  )

  // Not authenticated → login page
  // Exception: if ADMIN_PASSWORD not set on server, auth.authenticated = false
  // but the server allows all requests (dev mode). We show login only if server
  // explicitly said "authenticated: false" with a valid /auth/me response.
  return (
    <Routes>
      <Route path="/login" element={
        auth.authenticated ? <Navigate to="/" replace /> : <Login />
      } />
      <Route path="/*" element={
        <div style={{ display:'flex', height:'100vh', overflow:'hidden', background:'var(--bg)' }}>
          <Sidebar />
          <main style={{ flex:1, display:'flex', flexDirection:'column', overflow:'hidden', minWidth:0 }}>
            <div style={{ flex:1, overflowY:'auto' }}>
              <Routes>
                <Route path="/"              element={<Dashboard />} />
                <Route path="/cockpit"       element={<Cockpit />} />
                <Route path="/systems"       element={<Systems />} />
                <Route path="/agents"        element={<Agents />} />
                <Route path="/policies"      element={<Policies />} />
                <Route path="/jobs"          element={<Jobs />} />
                <Route path="/snapshots"     element={<Snapshots />} />
                <Route path="/restore-tests" element={<RestoreTests />} />
                <Route path="/repositories"  element={<Repositories />} />
                <Route path="/evidence"      element={<Evidence />} />
                <Route path="/alerts"        element={<Alerts />} />
                <Route path="/settings"      element={<Settings />} />
              </Routes>
            </div>
          </main>
        </div>
      } />
    </Routes>
  )
}

export default function App() {
  return (
    <BrowserRouter basename="/ui">
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </BrowserRouter>
  )
}
