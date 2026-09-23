"""Deterministic, explainable recommendation ranking over the project dataset."""

from __future__ import annotations

import re
from collections.abc import Iterable
from typing import Any

from .data_loader import DataBundle
from .models import Recommendation, RecommendationResponse, SkillGap


class RecommendationEngine:
    def __init__(self, data: DataBundle):
        self.data = data

    def recommend(self, employee_id: str, limit: int = 3) -> RecommendationResponse:
        employee = _find_by_id(self.data.employees, employee_id)
        if employee is None:
            raise ValueError(f"Employee not found: {employee_id}")
        current_level = _text(employee, "grade", "level", "current_grade", default="Unknown")
        target_level = _target_level(employee, current_level)
        gaps = self._skill_gaps(employee, current_level, target_level)
        history = [row for row in self.data.activity_history if _row_employee_id(row) == employee_id]
        recommendations = self._rank_events(employee, gaps, history, current_level, target_level)
        return RecommendationResponse(employee_id, current_level, target_level, gaps, recommendations[: max(0, limit)])

    def _skill_gaps(self, employee: dict[str, Any], current_level: str, target_level: str) -> list[SkillGap]:
        current_skills = _skill_map(employee)
        result: list[SkillGap] = []
        for definition in self.data.skills:
            skill_id = _id(definition, "skill_id", "id", "code", fallback=_text(definition, "name", "title", default="unknown"))
            skill_name = _text(definition, "name", "title", default=skill_id)
            current = _number(current_skills.get(skill_id), default=_number(current_skills.get(skill_name), default=0))
            if not current and skill_id in current_skills:
                current = _number(current_skills[skill_id])
            required = _required_level(definition, target_level, current_level)
            gap = max(0.0, required - current)
            if gap > 0:
                result.append(SkillGap(skill_id, skill_name, current, required, gap, _bool(definition, "critical", "is_critical")))
        return sorted(result, key=lambda item: (-item.critical, -item.gap, item.skill_id))

    def _rank_events(self, employee: dict[str, Any], gaps: list[SkillGap], history: list[dict[str, Any]], current_level: str, target_level: str) -> list[Recommendation]:
        by_key = {gap.skill_id.lower(): gap for gap in gaps} | {gap.skill_name.lower(): gap for gap in gaps}
        completed = {_event_key(row) for row in history if _is_completed(row)}
        ranked: list[Recommendation] = []
        for event in self.data.events:
            event_id = _id(event, "event_id", "activity_id", "id", fallback="")
            skill_ref = _text(event, "skill_id", "skill", "skill_code", "skill_name", default="").lower()
            gap = by_key.get(skill_ref)
            if gap is None:
                gap = next((item for key, item in by_key.items() if skill_ref and (skill_ref in key or key in skill_ref)), None)
            if gap is None:
                continue
            event_key = event_id.lower()
            completed_before = event_key in completed
            coverage = min(gap.gap, _number(event.get("gain", event.get("level_gain", 1)), default=1))
            role_match = _event_matches_role(event, employee)
            score = round((coverage / gap.gap) * 50 + (15 if gap.critical else 0) + (15 if role_match else 0) + (8 if not completed_before else -12) + _availability_score(event), 4)
            title = _text(event, "title", "name", "activity", default=event_id or gap.skill_name)
            reason = f"Закрывает {coverage:g} из {gap.gap:g} gap по навыку {gap.skill_name}."
            ranked.append(Recommendation(event_id, title, reason, gap.skill_name, gap.current_level, gap.required_level, coverage, score, {"critical": gap.critical, "role_match": role_match, "completed_before": completed_before, "current_level": current_level, "target_level": target_level}))
        return sorted(ranked, key=lambda item: (-item.priority, item.event_id, item.title))


def _find_by_id(rows: Iterable[dict[str, Any]], wanted: str) -> dict[str, Any] | None:
    return next((row for row in rows if _id(row, "employee_id", "id", "user_id", fallback="") == wanted), None)


def _skill_map(employee: dict[str, Any]) -> dict[str, Any]:
    values = employee.get("skills", employee.get("skill_levels", {}))
    if isinstance(values, dict):
        return values
    if isinstance(values, list):
        result = {}
        for row in values:
            if isinstance(row, dict):
                key = _id(row, "skill_id", "id", "code", fallback=_text(row, "name", default=""))
                result[key] = row.get("current_level", row.get("level", row.get("value", 0)))
        return result
    return {}


def _required_level(definition: dict[str, Any], target: str, current: str) -> float:
    for key in ("required_level", "target_level", "next_level"):
        if key in definition:
            return _number(definition[key])
    levels = definition.get("levels", definition.get("requirements", {}))
    if isinstance(levels, dict):
        for key in (target, current, "default"):
            if key in levels:
                value = levels[key]
                return _number(value.get("required", value.get("level", value)) if isinstance(value, dict) else value)
    return 0


def _target_level(employee: dict[str, Any], current: str) -> str:
    goal = employee.get("career_goal", employee.get("target_role", employee.get("career_target")))
    if isinstance(goal, dict):
        return _text(goal, "grade", "level", "title", default=current)
    return str(goal or current)


def _event_matches_role(event: dict[str, Any], employee: dict[str, Any]) -> bool:
    role = _text(employee, "role", "job_title", default="").lower()
    allowed = event.get("roles", event.get("target_roles", event.get("role")))
    if not allowed or not role:
        return False
    values = allowed if isinstance(allowed, list) else [allowed]
    return any(str(value).lower() in role or role in str(value).lower() for value in values)


def _availability_score(event: dict[str, Any]) -> float:
    status = _text(event, "status", "availability", default="").lower()
    return 5 if status in {"available", "open", "active", "available now"} else 0


def _row_employee_id(row: dict[str, Any]) -> str:
    return _id(row, "employee_id", "user_id", "employee", fallback="")


def _event_key(row: dict[str, Any]) -> str:
    return _id(row, "event_id", "activity_id", "id", fallback="").lower()


def _is_completed(row: dict[str, Any]) -> bool:
    return _text(row, "status", "outcome", default="").lower() in {"completed", "complete", "done", "finished"}


def _id(row: dict[str, Any], *keys: str, fallback: str) -> str:
    return str(next((row[key] for key in keys if row.get(key) not in (None, "")), fallback))


def _text(row: dict[str, Any], *keys: str, default: str) -> str:
    return str(next((row[key] for key in keys if row.get(key) not in (None, "")), default))


def _number(value: Any, default: float = 0) -> float:
    if isinstance(value, bool):
        return float(value)
    if isinstance(value, (int, float)):
        return float(value)
    match = re.search(r"-?\d+(?:\.\d+)?", str(value or ""))
    return float(match.group()) if match else default


def _bool(row: dict[str, Any], *keys: str) -> bool:
    return any(row.get(key) is True or str(row.get(key, "")).lower() in {"true", "yes", "1"} for key in keys)
