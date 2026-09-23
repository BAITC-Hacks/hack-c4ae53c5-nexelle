"""Pure, idempotent completion transition."""

from __future__ import annotations

import math
from dataclasses import dataclass, replace
from datetime import date

from .models import Activity, Employee, Event, SkillGain, Status


def validate_maximum(maximum: float) -> None:
    if isinstance(maximum, bool) or not math.isfinite(maximum) or maximum <= 0:
        raise ValueError("absolute_max_skill_level must be finite and positive")


def effective_gain(current: float, development: SkillGain, maximum: float) -> float:
    validate_maximum(maximum)
    values = (current, development.gain, development.max_level)
    if any(not math.isfinite(v) or v < 0 for v in values):
        raise ValueError("Skill levels and gains must be finite and nonnegative")
    if current > maximum:
        raise ValueError("Current level exceeds absolute maximum")
    if current >= development.max_level:
        return 0.0
    return min(current + development.gain, development.max_level, maximum) - current


@dataclass(frozen=True)
class CompletionResult:
    employee: Employee
    history: tuple[Activity, ...]
    applied: bool


def complete_event(
    employee: Employee,
    event: Event,
    history: tuple[Activity, ...],
    *,
    completed_on: date,
    completion_id: str,
    absolute_max_skill_level: float,
) -> CompletionResult:
    """Caller must atomically persist both employee and history."""
    validate_maximum(absolute_max_skill_level)
    if not completion_id.strip():
        raise ValueError("completion_id is required")
    for activity in history:
        if activity.completion_id == completion_id:
            if (activity.employee_id, activity.event_id, activity.status, activity.occurred_on) != (
                employee.employee_id,
                event.event_id,
                Status.COMPLETED,
                completed_on,
            ):
                raise ValueError("completion_id belongs to another completion")
            return CompletionResult(employee, history, False)
    if event.event_id != "EV_036" and any(
        a.employee_id == employee.employee_id
        and a.event_id == event.event_id
        and a.status == Status.COMPLETED
        for a in history
    ):
        return CompletionResult(employee, history, False)
    if len({g.skill_id for g in event.developed_skills}) != len(event.developed_skills):
        raise ValueError("Duplicate developed skill")
    skills = dict(employee.skills)
    for development in event.developed_skills:
        current = skills.get(development.skill_id, 0.0)
        skills[development.skill_id] = current + effective_gain(
            current, development, absolute_max_skill_level
        )
    activity = Activity(
        employee.employee_id, event.event_id, Status.COMPLETED, completed_on, completion_id
    )
    return CompletionResult(replace(employee, skills=skills), (*history, activity), True)
