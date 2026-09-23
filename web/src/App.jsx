import { useEffect, useState } from 'react'
import LanguageSwitch from './components/LanguageSwitch'
import RoleSwitch from './components/RoleSwitch'
import { createTranslator } from './i18n/translations'
import EmployeeDashboard from './pages/EmployeeDashboard'
import CareerPath from './pages/CareerPath'
import HRDashboard from './pages/HRDashboard'
import { completeActivity, DEMO_EMPLOYEE_ID, getEmployeeProfile, getHRDashboard } from './services/api'

function LoadingState() {
  return <div className="loading-state"><span className="loading-orb" /><p>Career Quest</p></div>
}

function ErrorState({ t, onRetry }) {
  return <div className="error-state"><p>{t('loadError')}</p><button className="button button--primary" onClick={onRetry} type="button">{t('retry')}</button></div>
}

export default function App() {
  const [locale, setLocale] = useState('ru')
  const [role, setRole] = useState('employee')
  const [view, setView] = useState('dashboard')
  const [employee, setEmployee] = useState(null)
  const [recommendations, setRecommendations] = useState([])
  const [hrData, setHrData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [completingId, setCompletingId] = useState(null)
  const [refreshing, setRefreshing] = useState(false)
  const [toast, setToast] = useState('')
  const t = createTranslator(locale)

  const loadEmployee = async () => {
    setLoading(true)
    setError(false)
    try {
      const profile = await getEmployeeProfile(DEMO_EMPLOYEE_ID, locale)
      setEmployee(profile.employee)
      setRecommendations(profile.recommendations)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }

  const loadHR = async () => {
    setLoading(true)
    setError(false)
    try {
      setHrData(await getHRDashboard())
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  useEffect(() => {
    if (role === 'employee') loadEmployee()
    else loadHR()
  }, [role, locale])

  const changeRole = (nextRole) => {
    setRole(nextRole)
    setView('dashboard')
    setToast('')
  }

  const handleComplete = async (eventId) => {
    setCompletingId(eventId)
    setToast('')
    try {
      const result = await completeActivity(DEMO_EMPLOYEE_ID, eventId, locale)
      setEmployee(result.employee)
      setRefreshing(true)
	  setRecommendations(result.recommendations)
      setToast(t('completedSuccess'))
      window.setTimeout(() => setToast(''), 4500)
    } catch {
      setToast(t('loadError'))
    } finally {
      setRefreshing(false)
      setCompletingId(null)
    }
  }

  const retry = () => role === 'employee' ? loadEmployee() : loadHR()
  const isEmployee = role === 'employee'

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand-lockup"><span className="brand-mark">CQ</span><div><strong>{t('brand')}</strong><span>{t('tagline')}</span></div></div>
        <div className="sidebar__label">{t('demoMode')}</div>
        {isEmployee ? (
          <nav className="sidebar-nav" aria-label="Main navigation">
            <button className={view === 'dashboard' ? 'is-active' : ''} onClick={() => setView('dashboard')} type="button"><span>▦</span>{t('dashboard')}</button>
            <button className={view === 'path' ? 'is-active' : ''} onClick={() => setView('path')} type="button"><span>↗</span>{t('careerPath')}</button>
          </nav>
        ) : <div className="hr-nav-label"><span>▦</span>{t('hrDashboard')}</div>}

        <div className="demo-flow">
          <p>{t('demoJourney')}</p>
          {isEmployee ? <ol>
            <li>{t('profile')}</li><li>{t('goal')}</li><li>{t('gaps')}</li><li>{t('aiRecommendation')}</li><li>{t('whyThis')}</li><li>{t('updatedProgress')}</li>
          </ol> : <ol>
            <li>{t('hrDashboard')}</li><li>{t('commonSkillGaps')}</li><li>{t('activityParticipation')}</li><li>{t('withoutNextStep')}</li>
          </ol>}
        </div>
        <div className="sidebar__footer">© 2026 Career Quest</div>
      </aside>

      <main className="main-content">
        <header className="topbar">
          <span className="topbar__context">{isEmployee ? t('employee') : t('hr')} <i /> {t('demoMode')}</span>
          <div className="topbar__controls"><RoleSwitch role={role} onChange={changeRole} t={t} /><LanguageSwitch locale={locale} onChange={setLocale} /></div>
        </header>

        <div className="content-area">
          {loading && <LoadingState />}
          {!loading && error && <ErrorState t={t} onRetry={retry} />}
          {!loading && !error && isEmployee && employee && (view === 'dashboard'
            ? <EmployeeDashboard employee={employee} recommendations={recommendations} t={t} onComplete={handleComplete} completingId={completingId} refreshing={refreshing} onViewPath={() => setView('path')} />
            : <CareerPath employee={employee} recommendations={recommendations} t={t} onComplete={handleComplete} completingId={completingId} />
          )}
          {!loading && !error && !isEmployee && hrData && <HRDashboard data={hrData} t={t} />}
        </div>
      </main>
      {toast && <div className="toast" role="status"><span>✓</span>{toast}</div>}
    </div>
  )
}
