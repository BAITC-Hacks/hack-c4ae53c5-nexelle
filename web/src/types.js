/**
 * Frontend contract for the demo adapter. Replace this boundary with the real
 * backend DTOs when they are available; components only consume this shape.
 *
 * CareerPath: { employee, careerGoal, skills[], recommendations[] }
 * Recommendation: { id, title, format, skill, current, required, gap,
 *   critical, availability, explanation, evidence[], scoreBreakdown? }
 * HRDashboard: { totals, skillGaps[], participation[], withoutNextStep }
 */
export const contract = Object.freeze({
  careerPath: ['employee', 'careerGoal', 'skills', 'recommendations'],
  recommendation: ['id', 'title', 'format', 'skill', 'current', 'required', 'gap', 'critical', 'availability', 'explanation', 'evidence'],
  hrDashboard: ['totals', 'skillGaps', 'participation', 'withoutNextStep'],
});
