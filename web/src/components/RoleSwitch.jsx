export default function RoleSwitch({ role, onChange, t }) {
  return (
    <div className="role-switch" aria-label={t('demoMode')}>
      <button type="button" className={role === 'employee' ? 'is-active' : ''} onClick={() => onChange('employee')}>{t('employee')}</button>
      <button type="button" className={role === 'hr' ? 'is-active' : ''} onClick={() => onChange('hr')}>{t('hr')}</button>
    </div>
  )
}
