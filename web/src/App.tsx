import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Sidebar } from './components/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { Systems } from './pages/Systems'
import { Jobs } from './pages/Jobs'
import { Snapshots } from './pages/Snapshots'
import { Policies } from './pages/Policies'
import { Repositories } from './pages/Repositories'

export default function App() {
  return (
    <BrowserRouter>
      <div style={{ display: 'flex', minHeight: '100vh' }}>
        <Sidebar />
        <main style={{ flex: 1, overflowY: 'auto' }}>
          <Routes>
            <Route path="/"             element={<Dashboard />} />
            <Route path="/systems"      element={<Systems />} />
            <Route path="/jobs"         element={<Jobs />} />
            <Route path="/snapshots"    element={<Snapshots />} />
            <Route path="/policies"     element={<Policies />} />
            <Route path="/repositories" element={<Repositories />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}
