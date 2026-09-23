export default function LanguageSwitch({ locale, onChange }) {
  return (
    <div className="language-switch" aria-label="Language">
      <button type="button" className={locale === 'ru' ? 'is-active' : ''} onClick={() => onChange('ru')}>RU</button>
      <span>/</span>
      <button type="button" className={locale === 'kk' ? 'is-active' : ''} onClick={() => onChange('kk')}>KZ</button>
    </div>
  )
}
