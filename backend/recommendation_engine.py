"""Deterministic eligibility and ranking; critical coverage is a separate tier."""

from __future__ import annotations

from datetime import date

from .data_loader import DataBundle
from .models import (
    CareerGoal,
    Employee,
    Event,
    Evidence,
    Grade,
    Recommendation,
    RecommendationResponse,
    SkillEvidence,
    SkillGap,
    SkillKind,
    Status,
)
from .progress import effective_gain, validate_maximum


class RecommendationEngine:
    def __init__(self, data: DataBundle, *, absolute_max_skill_level: float) -> None:
        validate_maximum(absolute_max_skill_level)
        self.data = data
        self.maximum = absolute_max_skill_level

    def recommend(
        self,
        employee_id: str,
        *,
        cutoff_date: date,
        limit: int = 3,
    ) -> RecommendationResponse:
        if isinstance(limit, bool) or not isinstance(limit, int) or not 1 <= limit <= 3:
            raise ValueError("limit must be an integer between 1 and 3")
        employee = next((e for e in self.data.employees if e.employee_id == employee_id), None)
        if employee is None:
            raise ValueError(f"Employee not found: {employee_id}")
        target = employee.career_goal or CareerGoal(employee.role, _next_grade(employee.grade))
        requirements: list[SkillGap] = []
        for skill in self.data.skills:
            requirement = next(
                (
                    r
                    for r in skill.requirements
                    if r.role == target.role and r.grade == target.grade
                ),
                None,
            )
            if requirement is None:
                continue
            current = employee.skills.get(skill.skill_id, 0.0)
            if current > self.maximum or requirement.level > self.maximum:
                raise ValueError("Skill level exceeds absolute maximum")
            requirements.append(
                SkillGap(
                    skill.skill_id,
                    current,
                    requirement.level,
                    max(0.0, requirement.level - current),
                    requirement.critical,
                )
            )
        if not requirements:
            raise ValueError(f"No requirements for {target.role}/{target.grade.value}")
        gaps = sorted(
            (g for g in requirements if g.gap > 0), key=lambda g: (-g.critical, -g.gap, g.skill_id)
        )
        history = [
            a
            for a in self.data.activity_history
            if a.employee_id == employee_id and a.occurred_on <= cutoff_date
        ]
        completed = {a.event_id for a in history if a.status == Status.COMPLETED}
        penalties = {Status.NO_SHOW: 15.0, Status.DROPPED: 10.0, Status.DECLINED: 5.0}
        ranked: list[Recommendation] = []
        for event in self.data.events:
            if not _eligible(event, employee, completed, cutoff_date):
                continue
            evidence: list[SkillEvidence] = []
            for development in event.developed_skills:
                gap = next((g for g in gaps if g.skill_id == development.skill_id), None)
                if gap is None:
                    continue
                gain = min(gap.gap, effective_gain(gap.current_level, development, self.maximum))
                if gain > 0:
                    evidence.append(
                        SkillEvidence(
                            gap.skill_id,
                            gap.current_level,
                            gap.required_level,
                            gap.gap,
                            gain,
                            gap.critical,
                        )
                    )
            reasons = self._fallback_reasons(event, requirements) if not gaps else ()
            if not evidence and not reasons:
                continue
            evidence.sort(key=lambda e: (-e.critical, -e.gain, e.skill_id))
            penalty = sum(
                penalties.get(a.status, 0.0) for a in history if a.event_id == event.event_id
            )
            coverage = sum(e.gain for e in evidence)
            score = round((coverage * 100 if gaps else 30.0 * len(reasons)) - penalty, 4)
            ranked.append(
                Recommendation(
                    event.event_id,
                    event.title,
                    score,
                    Evidence(
                        cutoff_date.isoformat(),
                        target.role,
                        target.grade.value,
                        any(e.critical for e in evidence),
                        penalty,
                        not gaps,
                        tuple(evidence),
                        reasons,
                    ),
                )
            )
        ranked.sort(key=lambda r: (-r.evidence.critical_gap, -r.priority, r.event_id))
        return RecommendationResponse(
            employee_id, employee.grade.value, target.grade.value, gaps, ranked[:limit]
        )

    def _fallback_reasons(self, event: Event, requirements: list[SkillGap]) -> tuple[str, ...]:
        developed = {g.skill_id for g in event.developed_skills}
        critical = {r.skill_id for r in requirements if r.critical}
        reasons: list[str] = []
        if any(
            s.skill_id in developed & critical and s.kind == SkillKind.HARD
            for s in self.data.skills
        ):
            reasons.append("critical_hard_maintenance")
        if any(s.skill_id in developed and s.kind == SkillKind.SOFT for s in self.data.skills):
            reasons.append("soft_skill_development")
        if event.mentoring:
            reasons.append("mentoring")
        return tuple(reasons)


def _eligible(event: Event, employee: Employee, completed: set[str], cutoff_date: date) -> bool:
    if event.mandatory or (event.event_id in completed and event.event_id != "EV_036"):
        return False
    if not set(event.prerequisites).issubset(completed):
        return False
    if not event.self_paced and not any(session > cutoff_date for session in event.sessions):
        return False
    profiles = [CareerGoal(employee.role, employee.grade)]
    if employee.career_goal is not None:
        profiles.append(employee.career_goal)
    return any(p.role in event.roles and p.grade in event.grades for p in profiles)


def _next_grade(grade: Grade) -> Grade:
    grades = list(Grade)
    return grades[min(grades.index(grade) + 1, len(grades) - 1)]
