"""Offline reference implementation. The running application uses Go exclusively."""

from __future__ import annotations

from datetime import date
from typing import Any

from .data_loader import DataBundle
from .models import Event, Recommendation, RecommendationResponse, SkillGap

ABSOLUTE_MAX_SKILL_LEVEL = 5
GRADES = ("Junior", "Middle", "Senior", "Lead")


def effective_gain(current: int, gain: int, maximum: int) -> int:
    if current >= maximum:
        return 0
    return max(0, min(current + gain, maximum, ABSOLUTE_MAX_SKILL_LEVEL) - current)


class RecommendationEngine:
    def __init__(self, data: DataBundle) -> None:
        self.data = data

    def recommend(
        self, employee_id: str, *, cutoff_date: date, limit: int = 3
    ) -> RecommendationResponse:
        if type(limit) is not int or not 1 <= limit <= 3:
            raise ValueError("limit must be between 1 and 3")
        employee = next((e for e in self.data.employees if e["employee_id"] == employee_id), None)
        if employee is None:
            raise ValueError(f"Unknown employee {employee_id}")
        grade = employee["grade"]
        goal = employee.get("career_goal")
        target_role = goal["target_role"] if goal else employee["role"]
        target_grade = goal["target_grade"] if goal else GRADES[min(GRADES.index(grade) + 1, 3)]
        profile = next(
            (
                p
                for p in self.data.role_profiles
                if p["role"] == target_role and p["grade"] == target_grade
            ),
            None,
        )
        if profile is None:
            raise ValueError("Target role profile is missing")
        events = {e.event_id: e for e in self.data.events}
        history = sorted(
            (
                h
                for h in self.data.activity_history
                if h["employee_id"] == employee_id and date.fromisoformat(h["date"]) <= cutoff_date
            ),
            key=lambda h: (h["date"], h["record_id"]),
        )
        skills = dict(employee["skills"])
        review = date.fromisoformat(employee["last_review_date"])
        for activity in history:
            if activity["status"] == "completed" and date.fromisoformat(activity["date"]) > review:
                for development in events[activity["event_id"]].develops_skills:
                    current = skills.get(development.skill_id, 0)
                    skills[development.skill_id] = current + effective_gain(
                        current, development.gain, development.max_level
                    )
        definitions = {s["skill_id"]: s for s in self.data.skills}
        all_gaps = {
            skill_id: SkillGap(
                skill_id,
                definitions[skill_id]["name"],
                skills.get(skill_id, 0),
                required,
                max(0, required - skills.get(skill_id, 0)),
                skill_id in profile["critical_skills"],
            )
            for skill_id, required in profile["required_skills"].items()
        }
        gaps = sorted(
            (g for g in all_gaps.values() if g.gap > 0),
            key=lambda g: (-g.critical, -g.gap, g.skill_id),
        )
        completed = {h["event_id"] for h in history if h["status"] == "completed"}
        ranked: list[Recommendation] = []
        for event in self.data.events:
            if not self._eligible(event, employee, skills, completed, cutoff_date):
                continue
            evidence: list[dict[str, Any]] = []
            for development in event.develops_skills:
                gap = all_gaps.get(development.skill_id)
                if gap is None or gap.gap <= 0:
                    continue
                gain = min(
                    gap.gap,
                    effective_gain(int(gap.current_level), development.gain, development.max_level),
                )
                if gain > 0:
                    evidence.append(
                        {"skill_id": gap.skill_id, "gain": gain, "critical": gap.critical}
                    )
            reasons: list[str] = []
            if not gaps:
                developed = {g.skill_id for g in event.develops_skills}
                if any(
                    s in profile["critical_skills"] and definitions[s]["type"] == "hard"
                    for s in developed
                ):
                    reasons.append("critical_hard_maintenance")
                if any(definitions[s]["type"] == "soft" for s in developed):
                    reasons.append("soft_skill_development")
                if event.type == "mentoring":
                    reasons.append("mentoring")
            if not evidence and not reasons:
                continue
            relevant = [h for h in history if events[h["event_id"]].format == event.format]
            negatives = sum(h["status"] in {"no_show", "dropped", "declined"} for h in relevant)
            positives = sum(h["status"] == "completed" for h in relevant)
            gap_score = sum(e["gain"] * (2.5 if e["critical"] else 1) for e in evidence)
            goal_score = (
                5
                if target_role in event.target_roles and target_grade in event.target_grades
                else 2
            )
            next_session = min(
                (s for s in event.upcoming_sessions if s > cutoff_date), default=None
            )
            days = (next_session - cutoff_date).days if next_session else 0
            feasibility = 3 if event.format == "self_paced" or days <= 7 else 2 if days <= 30 else 1
            score = gap_score + goal_score + positives * 2 - negatives * 5 + feasibility
            primary = next(
                (g for g in gaps if any(e["skill_id"] == g.skill_id for e in evidence)), None
            )
            ranked.append(
                Recommendation(
                    event.event_id,
                    event.title,
                    "Поддержание и развитие навыков."
                    if reasons
                    else "Сокращает разрыв до целевого грейда.",
                    primary.skill_name if primary else "",
                    primary.current_level if primary else 0,
                    primary.required_level if primary else 0,
                    sum(e["gain"] for e in evidence),
                    float(score),
                    {
                        "critical": any(e["critical"] for e in evidence),
                        "gap_score": gap_score,
                        "fallback": bool(reasons),
                        "fallback_reasons": reasons,
                        "skills": evidence,
                        "penalty": negatives * 5,
                        "cutoff_date": cutoff_date.isoformat(),
                    },
                )
            )
        ranked.sort(
            key=lambda r: (
                -bool(r.factors["critical"]),
                -r.priority,
                -r.factors["gap_score"],
                r.event_id,
            )
        )
        return RecommendationResponse(employee_id, grade, target_grade, gaps, ranked[:limit])

    @staticmethod
    def _eligible(
        event: Event,
        employee: dict[str, Any],
        skills: dict[str, int],
        completed: set[str],
        cutoff: date,
    ) -> bool:
        if event.mandatory or (event.event_id in completed and event.event_id != "EV_036"):
            return False
        audiences = [(employee["role"], employee["grade"])]
        if employee.get("career_goal"):
            goal = employee["career_goal"]
            audiences.append((goal["target_role"], goal["target_grade"]))
        if not any(
            role in event.target_roles and grade in event.target_grades for role, grade in audiences
        ):
            return False
        if any(skills.get(skill, 0) < level for skill, level in event.prerequisites.items()):
            return False
        if not set(event.prerequisite_events).issubset(completed):
            return False
        return event.format == "self_paced" or any(
            session > cutoff for session in event.upcoming_sessions
        )
