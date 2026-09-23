import type { EventDTO, HRDTO, Locale, ProfileDTO, RecommendationDTO, SkillDTO } from './contracts.ts';

// This mapping is presentation only. Scores, order, gains and completion come from Go.
export function createClient(base = '', employeeId = 'E0001', fetcher: typeof fetch = fetch) {
  async function request<T>(path: string, role: 'employee' | 'hr', locale: Locale, init: RequestInit = {}): Promise<T> {
    const response = await fetcher(base + path + '?locale=' + locale, {
      ...init,
      headers: { 'Content-Type': 'application/json', 'X-Demo-Role': role, 'X-Employee-ID': employeeId },
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => null);
      throw new Error(payload?.error?.message ?? ('HTTP ' + response.status));
    }
    return response.json() as Promise<T>;
  }

  async function catalogs(locale: Locale, signal?: AbortSignal) {
    const [events, skills] = await Promise.all([
      request<EventDTO[]>('/api/events', 'employee', locale, { signal }),
      request<SkillDTO[]>('/api/skills', 'employee', locale, { signal }),
    ]);
    return { events, skills };
  }

  return {
    async employee(locale: Locale = 'ru', signal?: AbortSignal) {
      const [profile, catalog] = await Promise.all([
        request<ProfileDTO>('/api/v1/employees/' + encodeURIComponent(employeeId) + '/profile', 'employee', locale, { signal }),
        catalogs(locale, signal),
      ]);
      return mapProfile(profile, catalog.events, catalog.skills, locale);
    },
    async complete(eventId: string, locale: Locale = 'ru') {
      const { profile } = await request<{ profile: ProfileDTO }>(
        '/api/v1/employees/' + encodeURIComponent(employeeId) + '/complete', 'employee', locale,
        { method: 'POST', body: JSON.stringify({ eventId }) },
      );
      const catalog = await catalogs(locale);
      return mapProfile(profile, catalog.events, catalog.skills, locale);
    },
    async hr(locale: Locale = 'ru', signal?: AbortSignal) {
      const [data, catalog] = await Promise.all([
        request<HRDTO>('/api/v1/hr/analytics', 'hr', locale, { signal }), catalogs(locale, signal),
      ]);
      const names = new Map(catalog.skills.map(s => [s.skill_id, s.name]));
      return {
        total_employees: data.totalEmployees,
        employees_in_development: data.employeesInDevelopment,
        employees_without_recommendation: data.employeesWithoutRecommendation.length,
        skill_gaps: data.topSkillGaps.map(g => ({ skill: names.get(g.skillId) ?? g.skillId, employees: g.employeeCount })),
        activity_participation: data.participationByEvent.reduce((total, row) => ({
          completed: total.completed + row.completed, no_show: total.no_show + row.noShow,
          dropped: total.dropped + row.dropped, declined: total.declined + row.declined,
          total: total.total + row.registrations,
        }), { completed: 0, no_show: 0, dropped: 0, declined: 0, total: 0 }),
      };
    },
  };
}

function mapProfile(profile: ProfileDTO, events: EventDTO[], skills: SkillDTO[], locale: Locale) {
  const names = new Map(skills.map(s => [s.skill_id, s.name]));
  const catalog = new Map(events.map(e => [e.event_id, e]));
  return {
    employee: {
      employee_id: profile.employee.employeeId, name: profile.employee.fullName,
      role: profile.employee.role, grade: profile.employee.grade,
      career_goal: {
        target_role: profile.progress.target.role, target_grade: profile.progress.target.grade,
        target_date: profile.cutoffDate,
      },
      skills: profile.progress.skillGaps.map(g => ({
        name: names.get(g.skillId) ?? g.skillId, current_level: g.currentLevel,
        required_level: g.requiredLevel, gap: g.gap,
      })),
      history: profile.activities.map(a => ({
        activity_id: a.recordId, title: catalog.get(a.eventId)?.title ?? a.eventId,
        format: displayFormat(catalog.get(a.eventId)), date: a.date, status: a.status,
      })).reverse(),
      progress: {
        percentage: profile.progress.readinessPercent,
        completed_activities: profile.activities.filter(a => a.status === 'completed').length,
        total_activities: profile.activities.length,
      },
    },
    // Never re-sort: the critical tier has precedence over the numeric score.
    recommendations: profile.recommendations.map(r => mapRecommendation(r, catalog.get(r.eventId), names, locale)),
  };
}

function mapRecommendation(r: RecommendationDTO, event: EventDTO | undefined, names: Map<string, string>, locale: Locale) {
  const selfPaced = locale === 'kk' ? 'Өз қарқынымен' : 'В своём темпе';
  return {
    event_id: r.eventId, title: r.title,
    format: displayFormat(event),
    score_breakdown: { skill_gaps: r.score.gapScore, career_goal: r.score.goalMatchScore,
      activity_history: r.score.historyScore, availability: r.score.feasibilityScore },
    skill_gaps: r.evidence.skills.map(s => names.get(s.skillId) ?? s.skillId),
    expected_gain: r.evidence.skills.map(s => ({ skill: names.get(s.skillId) ?? s.skillId, level_delta: s.gain })),
    eligibility: { prerequisites: [
      ...Object.entries(event?.prerequisites ?? {}).map(([id, level]) => (names.get(id) ?? id) + ' ≥ ' + level),
      ...(event?.prerequisite_events ?? []),
    ] },
    availability: { status: 'available', date: r.nextSessionDate ?? selfPaced },
    explanation: {
      why: r.explanation,
      evidence: r.evidence.skills.map(s => ({
        factor: 'skill_gaps',
        detail: (names.get(s.skillId) ?? s.skillId) + ': ' + s.currentLevel + ' / ' + s.requiredLevel + ', +' + s.gain,
      })),
    },
  };
}

function displayFormat(event: EventDTO | undefined): string {
  return event?.type === 'course' || event?.type === 'certification' ? 'Course' :
    event?.type === 'workshop' ? 'Workshop' : event?.type === 'mentoring' ? 'Mentoring' : 'Event';
}

export const api = createClient();
