const initial = {
  employee: { id: 'demo-employee', name: 'Алия Садыкова', role: 'Product Designer', grade: 'Middle', location: 'Алматы' },
  careerGoal: { title: 'Senior Product Designer', grade: 'Senior' },
  skills: [
    { id: 'research', name: 'User Research', current: 3, required: 4, critical: true },
    { id: 'systems', name: 'Systems Thinking', current: 2, required: 4, critical: true },
    { id: 'facilitation', name: 'Workshop Facilitation', current: 3, required: 4, critical: false },
  ],
  recommendations: [
    { id: 'activity-1', title: 'Designing for complex systems', format: 'Self-paced', skill: 'Systems Thinking', current: 2, required: 4, gap: 2, critical: true, availability: 'Available now', explanation: 'Builds the systems perspective required for your Senior Product Designer path.', evidence: ['Systems Thinking is 2/4; target level is 4.', 'This is a critical skill for your career goal.', 'You completed a similar self-paced module and made progress.'], scoreBreakdown: [{ label: 'Goal alignment', value: '+35' }, { label: 'Skill gap coverage', value: '+30' }, { label: 'Learning history', value: '+15' }] },
    { id: 'activity-2', title: 'Research synthesis lab', format: 'Live session', skill: 'User Research', current: 3, required: 4, gap: 1, critical: true, availability: 'Next session · 5 Oct, 18:00', explanation: 'Practice turning interview findings into product decisions with a facilitator.', evidence: ['User Research is 3/4; target level is 4.', 'This is a critical skill for your career goal.', 'A session is available in your time zone.'] },
    { id: 'activity-3', title: 'Facilitate a product workshop', format: 'Practice', skill: 'Workshop Facilitation', current: 3, required: 4, gap: 1, critical: false, availability: 'Available now', explanation: 'A guided practice session to strengthen your workshop facilitation.', evidence: ['Workshop Facilitation is 3/4; target level is 4.', 'The practice format matches your recent learning history.', 'This activity is available now.'] },
  ],
};

const hr = {
  totals: { employees: 48, withCareerGoal: 36, withoutCareerGoal: 12 },
  skillGaps: [{ name: 'Systems Thinking', count: 18 }, { name: 'User Research', count: 13 }, { name: 'Stakeholder Alignment', count: 9 }, { name: 'Workshop Facilitation', count: 7 }],
  participation: [{ name: 'Completed', count: 76, color: 'green' }, { name: 'No show', count: 9, color: 'amber' }, { name: 'Dropped', count: 7, color: 'slate' }, { name: 'Declined', count: 8, color: 'rose' }],
  withoutNextStep: { count: 4, reasons: ['No eligible activity', 'No available session'] },
};

const STORAGE_KEY = 'career-quest-demo-state-v1';
export function createMockApi() {
  let state;
  try { state = JSON.parse(localStorage.getItem(STORAGE_KEY)) || structuredClone(initial); }
  catch { state = structuredClone(initial); }
  const save = () => localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  return {
    async getCareerPath() { await delay(220); return structuredClone(state); },
    async getHRDashboard() { await delay(220); return structuredClone(hr); },
    async completeActivity(id) {
      await delay(300);
      const recommendation = state.recommendations.find(item => item.id === id);
      if (!recommendation) throw new Error('Activity is no longer available.');
      const skill = state.skills.find(item => item.name === recommendation.skill);
      const before = skill.current;
      if (skill.current < skill.required) skill.current += 1;
      state.recommendations = state.recommendations.filter(item => item.id !== id);
      const completed = { title: recommendation.title, skill: recommendation.skill, before, after: skill.current, maxed: before === skill.current };
      save();
      return completed;
    },
    async resetDemo() { state = structuredClone(initial); save(); return structuredClone(state); },
  };
}
function delay(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }
