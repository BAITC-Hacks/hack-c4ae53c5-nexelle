"""Typed, dependency-free models used by the analysis layer."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from datetime import date
from typing import Any


@dataclass(frozen=True)
class EventSkillGain:
    skill_id: str
    gain: int
    max_level: int


@dataclass(frozen=True)
class Event:
    event_id: str
    title: str
    description: str
    type: str
    format: str
    duration_hours: float
    mandatory: bool
    target_roles: tuple[str, ...]
    target_grades: tuple[str, ...]
    develops_skills: tuple[EventSkillGain, ...]
    prerequisites: dict[str, int]
    upcoming_sessions: tuple[date, ...]
    prerequisite_events: tuple[str, ...] = ()


@dataclass(frozen=True)
class SkillGap:
    skill_id: str
    skill_name: str
    current_level: float
    required_level: float
    gap: float
    critical: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class Recommendation:
    event_id: str
    title: str
    reason: str
    skill_gap: str
    current_skill_level: float
    required_skill_level: float
    gap_covered: float
    priority: float
    factors: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class RecommendationResponse:
    employee_id: str
    current_level: str
    target_level: str
    skill_gaps: list[SkillGap]
    recommendations: list[Recommendation]

    def to_dict(self) -> dict[str, Any]:
        result = asdict(self)
        result["skill_gaps"] = [item.to_dict() for item in self.skill_gaps]
        result["recommendations"] = [item.to_dict() for item in self.recommendations]
        return result


@dataclass(frozen=True)
class AIExplanation:
    summary: str
    why: str
    next_steps: list[str]

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)
