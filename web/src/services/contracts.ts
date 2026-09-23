export interface SkillDTO { skill_id: string; name: string; type: 'hard' | 'soft' }
export interface EventDTO {
  event_id: string; title: string; type: string; format: string;
  target_roles: string[]; target_grades: string[];
  develops_skills: { skill_id: string; gain: number; max_level: number }[];
  prerequisites: Record<string, number>; prerequisite_events?: string[];
  upcoming_sessions: string[];
}
export interface SkillEvidence {
  skillId: string; currentLevel: number; requiredLevel: number; gain: number; isCritical: boolean;
}
export interface RecommendationDTO {
  eventId: string; title: string; format: string; nextSessionDate?: string; explanation: string;
  score: { gapScore: number; goalMatchScore: number; historyScore: number; feasibilityScore: number; total: number };
  evidence: {
    skills: SkillEvidence[]; criticalGap: boolean; fallback: boolean; fallbackReasons: string[];
    history: { format: string; completedCount: number; negativeCount: number };
  };
}
export interface ProfileDTO {
  cutoffDate: string;
  employee: { employeeId: string; fullName: string; role: string; grade: string };
  progress: {
    target: { role: string; grade: string; source: string }; readinessPercent: number;
    effectiveSkills: Record<string, number>;
    skillGaps: { skillId: string; currentLevel: number; requiredLevel: number; gap: number; isCritical: boolean }[];
  };
  activities: { recordId: string; eventId: string; date: string; status: string }[];
  recommendations: RecommendationDTO[];
}
export interface HRDTO {
  totalEmployees: number; employeesInDevelopment: number;
  topSkillGaps: { skillId: string; employeeCount: number }[];
  employeesWithoutRecommendation: { employeeId: string; fullName: string; reason: string }[];
  participationByEvent: { completed: number; noShow: number; dropped: number; declined: number; registrations: number }[];
}
export type Locale = 'ru' | 'kk';
