import { api } from './api.js';
import { t } from './i18n.js';

const app = document.querySelector('#app');
let language = localStorage.getItem('career-quest-language') || 'ru';
let mode = 'employee';
let data = null;
let loading = true;
let error = '';
let completing = '';
let toast = '';
let expanded = new Set();

const escapeHtml = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
const percent = (current, required) => required > 0 ? Math.min(100, Math.round(current / required * 100)) : 0;

async function load() {
  loading = true; error = ''; render();
  try { data = mode === 'employee' ? await api.getCareerPath() : await api.getHRDashboard(); }
  catch (err) { error = err.message || 'Unable to load data'; }
  finally { loading = false; render(); }
}

function shell(content) {
  const label = t(language, mode);
  return `<div class="shell">
    <aside class="sidebar">
      <a class="brand" href="#" aria-label="Career Quest"><span class="brand-mark">cq</span><span>career<span class="brand-light">quest</span></span></a>
      <div class="workspace-label">WORKSPACE</div>
      <button class="side-link ${mode === 'employee' ? 'active' : ''}" data-mode="employee"><span class="side-icon">◉</span>${escapeHtml(t(language, 'employee'))}</button>
      <button class="side-link ${mode === 'hr' ? 'active' : ''}" data-mode="hr"><span class="side-icon">▦</span>${escapeHtml(t(language, 'hr'))}</button>
      <div class="sidebar-bottom"><div class="help-card"><span class="help-icon">✳</span><strong>Small steps,<br/>real growth.</strong><span>Make room for what’s next.</span></div><div class="side-foot">CAREER QUEST <span>·</span> MVP</div></div>
    </aside>
    <main class="main"><header class="topbar"><div class="crumb">Workspace <span>/</span> <b>${escapeHtml(label)}</b></div><div class="top-actions"><span class="demo-pill"><i></i>${escapeHtml(t(language, 'demo'))}</span><div class="divider"></div><button class="lang-switch" data-lang="ru" aria-pressed="${language === 'ru'}">RU</button><span class="lang-slash">/</span><button class="lang-switch" data-lang="kk" aria-pressed="${language === 'kk'}">KZ</button><button class="avatar" aria-label="Demo user">A</button></div></header>${content}</main>
    ${toast ? `<div class="toast"><span>✓</span>${escapeHtml(toast)}</div>` : ''}
  </div>`;
}

function render() {
  if (loading) { app.innerHTML = shell(`<section class="loading-state"><div class="loader"></div><span>${escapeHtml(t(language, 'loading'))}</span></section>`); bind(); return; }
  if (error) { app.innerHTML = shell(`<section class="error-card"><div class="error-icon">!</div><h2>${escapeHtml(error)}</h2><p>We couldn't load your career path.</p><button class="button button-dark" data-retry>${escapeHtml(t(language, 'retry'))}<span>↗</span></button></section>`); bind(); return; }
  app.innerHTML = shell(mode === 'employee' ? employeePage(data) : hrPage(data));
  bind();
}

function employeePage(path) {
  const employee = path.employee || {};
  const noGoal = !path.careerGoal;
  const isLead = employee.grade === 'Lead';
  const recommendations = (path.recommendations || []).slice(0, 3);
  const skills = path.skills || [];
  return `<div class="page-wrap">
    <div class="page-heading"><div><div class="eyebrow">${escapeHtml(t(language, 'profile'))} <span class="eyebrow-dot"></span> PERSONAL DEVELOPMENT</div><h1>Good morning, ${escapeHtml(employee.name?.split(' ')[0] || 'there')}<span class="wave">✳</span></h1><p class="page-subtitle">A clear view of where you are and what’s next.</p></div><button class="button button-quiet" data-reset title="Reset demo data">↺ <span>${escapeHtml(t(language, 'reset'))}</span></button></div>
    <section class="hero-card"><div class="hero-orbit orbit-one"></div><div class="hero-orbit orbit-two"></div><div class="hero-content"><div class="hero-kicker"><span class="sparkle">✳</span> YOUR CAREER DIRECTION</div><h2>${escapeHtml(path.careerGoal?.title || (isLead ? `${t(language, 'grade')}: Lead` : t(language, 'noGoal')))}</h2><p>${noGoal ? escapeHtml(isLead ? t(language, 'topLevelHint') : t(language, 'noGoalText')) : 'Build the capabilities that move your work forward.'}</p><div class="hero-meta"><span><i class="meta-icon">↗</i>${escapeHtml(employee.role || '—')}</span><span><i class="meta-icon">◈</i>${escapeHtml(employee.grade || '—')}</span><span><i class="meta-icon">⌖</i>${escapeHtml(employee.location || '—')}</span></div></div><div class="path-visual"><div class="path-label">YOUR NEXT CHAPTER</div><div class="path-node node-done"><span>✓</span><small>NOW</small></div><div class="path-line"></div><div class="path-node node-next"><span>✳</span><small>NEXT</small></div><div class="path-caption"><b>${escapeHtml(employee.grade || '')}</b><span>${escapeHtml(path.careerGoal?.grade || (isLead ? 'Lead' : 'Growth'))}</span></div></div></section>
    <div class="section-heading"><div><div class="eyebrow">01 <span class="eyebrow-dot"></span> CAPABILITY MAP</div><h2>${escapeHtml(t(language, 'gapTitle'))}</h2></div><span class="section-note">${skills.length} skills in focus <span>↗</span></span></div>
    ${skills.length ? `<section class="skill-grid">${skills.map(skillCard).join('')}</section>` : `<div class="empty-inline"><span>✳</span>${escapeHtml(t(language, 'noGaps'))}</div>`}
    <div class="section-heading recommendations-heading"><div><div class="eyebrow">02 <span class="eyebrow-dot"></span> CURATED FOR YOU</div><h2>${escapeHtml(t(language, 'nextSteps'))}</h2></div><span class="count-badge">${recommendations.length.toString().padStart(2, '0')} <span>STEPS</span></span></div>
    ${recommendations.length ? `<section class="recommendation-list">${recommendations.map(recommendationCard).join('')}</section>` : `<section class="empty-card"><div class="empty-symbol">✳</div><h3>${escapeHtml(t(language, 'noRecs'))}</h3><p>${escapeHtml(t(language, 'noRecsText'))}</p></section>`}
    <footer class="page-footer">A little progress each day adds up to big results. <span>✳</span></footer>
  </div>`;
}

function skillCard(skill) {
  const pct = percent(skill.current, skill.required);
  const gap = Math.max(0, skill.required - skill.current);
  return `<article class="skill-card"><div class="skill-top"><div class="skill-symbol">${skill.critical ? '✳' : '◌'}</div><span class="skill-tag ${skill.critical ? 'tag-critical' : ''}">${skill.critical ? escapeHtml(t(language, 'critical')) : 'IN PROGRESS'}</span></div><h3>${escapeHtml(skill.name)}</h3><div class="skill-values"><strong>${skill.current}<span> / ${skill.required}</span></strong><span>${pct}%</span></div><div class="progress-track"><div style="width:${pct}%"></div></div><div class="skill-foot"><span>${escapeHtml(t(language, 'current'))} <b>${skill.current}</b> <i>→</i> ${escapeHtml(t(language, 'required'))} <b>${skill.required}</b></span><span class="gap-label">${escapeHtml(t(language, 'gap'))} <b>${gap}</b></span></div></article>`;
}

function recommendationCard(item, index) {
  const open = expanded.has(item.id);
  return `<article class="recommendation-card"><div class="rec-index">0${index + 1}</div><div class="rec-body"><div class="rec-topline"><span class="rec-format"><span class="format-dot"></span>${escapeHtml(item.format)}</span>${item.critical ? `<span class="critical-mini">✳ ${escapeHtml(t(language, 'critical'))}</span>` : ''}<span class="rec-availability">◷ ${escapeHtml(item.availability || t(language, 'available'))}</span></div><h3>${escapeHtml(item.title)}</h3><div class="rec-detail-grid"><div><span>SKILL</span><b>${escapeHtml(item.skill)}</b></div><div><span>${escapeHtml(t(language, 'current'))}</span><b>${item.current} <i>/ ${item.required}</i></b></div><div><span>${escapeHtml(t(language, 'gap'))}</span><b class="gap-number">${item.gap}</b></div></div><button class="evidence-toggle" data-evidence="${escapeHtml(item.id)}" aria-expanded="${open}"><span class="evidence-icon">✧</span>${escapeHtml(t(language, 'why'))}<span class="chevron ${open ? 'turned' : ''}">⌄</span></button>${open ? `<div class="evidence-panel"><p>${escapeHtml(item.explanation || '')}</p><ul>${(item.evidence || []).map(fact => `<li><span>✓</span>${escapeHtml(fact)}</li>`).join('')}</ul>${item.scoreBreakdown?.length ? `<details class="score-details"><summary>${escapeHtml(t(language, 'score'))}</summary><div>${item.scoreBreakdown.map(row => `<span>${escapeHtml(row.label)}</span><b>${escapeHtml(row.value)}</b>`).join('')}</div></details>` : ''}</div>` : ''}</div><div class="rec-action"><button class="button button-dark complete-button" data-complete="${escapeHtml(item.id)}" ${completing === item.id ? 'disabled' : ''}>${completing === item.id ? '…' : escapeHtml(t(language, 'complete'))}<span>↗</span></button><small>${escapeHtml(item.availability || t(language, 'available'))}</small></div></article>`;
}

function hrPage(dashboard) {
  const totals = dashboard.totals || {};
  const participation = dashboard.participation || [];
  const maxParticipation = Math.max(1, ...participation.map(item => item.count));
  return `<div class="page-wrap hr-wrap"><div class="page-heading"><div><div class="eyebrow">TEAM OVERVIEW <span class="eyebrow-dot"></span> PEOPLE & GROWTH</div><h1>${escapeHtml(t(language, 'team'))}<span class="wave">✳</span></h1><p class="page-subtitle">A thoughtful snapshot of learning across your team.</p></div><span class="updated-label"><i></i> LIVE OVERVIEW</span></div>
    <section class="stats-grid"><article class="stat-card stat-dark"><span>TEAM SIZE</span><strong>${totals.employees ?? '—'}</strong><small>people in your team</small><div class="stat-decoration">✳</div></article><article class="stat-card"><span>CAREER DIRECTION</span><strong>${totals.withCareerGoal ?? '—'}<small> / ${totals.employees ?? '—'}</small></strong><div class="stat-bottom"><span>${escapeHtml(t(language, 'withGoal'))}</span><b>${totals.employees ? Math.round(totals.withCareerGoal / totals.employees * 100) : 0}%</b></div><div class="mini-track"><i style="width:${totals.employees ? Math.round(totals.withCareerGoal / totals.employees * 100) : 0}%"></i></div></article><article class="stat-card stat-warm"><span>ROOM TO GROW</span><strong>${totals.withoutCareerGoal ?? '—'}</strong><small>${escapeHtml(t(language, 'withoutGoal'))}</small><div class="stat-decoration">↗</div></article></section>
    <section class="hr-panels"><article class="panel gaps-panel"><div class="panel-heading"><div><span class="panel-overline">TEAM DEVELOPMENT</span><h2>${escapeHtml(t(language, 'commonGaps'))}</h2></div><span class="panel-icon">⌖</span></div><div class="gap-bars">${(dashboard.skillGaps || []).map((item, i) => `<div class="gap-row"><div class="gap-row-head"><span><i class="gap-rank">0${i + 1}</i>${escapeHtml(item.name)}</span><b>${item.count} <small>${escapeHtml(t(language, 'people'))}</small></b></div><div class="wide-track"><i style="width:${Math.min(100, item.count / Math.max(1, dashboard.skillGaps[0]?.count) * 100)}%"></i></div></div>`).join('')}</div></article><article class="panel participation-panel"><div class="panel-heading"><div><span class="panel-overline">LEARNING MOMENTUM</span><h2>${escapeHtml(t(language, 'participation'))}</h2></div><span class="panel-icon">◷</span></div><div class="participation-list">${participation.map(item => `<div class="participation-row"><div class="participation-label"><i class="legend-dot ${escapeHtml(item.color)}"></i><span>${escapeHtml(t(language, item.name === 'Completed' ? 'completedPlural' : item.name === 'No show' ? 'noShow' : item.name === 'Dropped' ? 'dropped' : 'declined'))}</span><b>${item.count}%</b></div><div class="wide-track"><i class="bar-${escapeHtml(item.color)}" style="width:${Math.min(100, item.count / maxParticipation * 100)}%"></i></div></div>`).join('')}</div><div class="participation-caption">Based on recorded activity outcomes</div></article></section>
    <section class="without-next"><div class="without-icon">↗</div><div class="without-copy"><span class="panel-overline">FOLLOW-UP</span><h2>${escapeHtml(t(language, 'withoutNext'))}</h2><p>${dashboard.withoutNextStep?.count ?? '—'} employees currently have no eligible next step.</p>${dashboard.withoutNextStep?.reasons?.length ? `<div class="reason-tags">${dashboard.withoutNextStep.reasons.map(reason => `<span>${escapeHtml(reason)}</span>`).join('')}</div>` : ''}</div><div class="without-count"><strong>${dashboard.withoutNextStep?.count ?? '—'}</strong><span>NEED A LOOK</span></div></section><footer class="page-footer">Support the next step. The growth follows. <span>✳</span></footer></div>`;
}

function bind() {
  app.querySelectorAll('[data-mode]').forEach(button => button.addEventListener('click', () => { mode = button.dataset.mode; data = null; load(); }));
  app.querySelectorAll('[data-lang]').forEach(button => button.addEventListener('click', () => { language = button.dataset.lang; localStorage.setItem('career-quest-language', language); render(); }));
  app.querySelectorAll('[data-retry]').forEach(button => button.addEventListener('click', load));
  app.querySelectorAll('[data-evidence]').forEach(button => button.addEventListener('click', () => { const id = button.dataset.evidence; expanded.has(id) ? expanded.delete(id) : expanded.add(id); render(); }));
  app.querySelectorAll('[data-complete]').forEach(button => button.addEventListener('click', async () => {
    completing = button.dataset.complete; render();
    try {
      const result = await api.completeActivity(completing);
      toast = result.maxed ? t(language, 'maxed') : `${result.skill}: ${result.before} → ${result.after}. ${t(language, 'progressUpdated')}`;
      completing = ''; await load(); setTimeout(() => { toast = ''; render(); }, 3600);
    } catch (err) { completing = ''; error = err.message; loading = false; render(); }
  }));
  app.querySelectorAll('[data-reset]').forEach(button => button.addEventListener('click', async () => { await api.resetDemo(); toast = ''; expanded.clear(); await load(); }));
}

load();
