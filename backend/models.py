"""Typed domain and result contracts."""

from __future__ import annotations

from dataclasses import asdict, dataclass
from datetime import date
from enum import Enum


class Grade(str, Enum):
    JUNIOR = "Junior"
    MIDDLE = "Middle"
    SENIOR = "Senior"
    LEAD = "Lead"


class Locale(str, Enum):
    RU = "ru"
    KK = "kk"


class SkillKind(str, Enum):
    HARD = "hard"
    SOFT = "soft"


class Status(str, Enum):
    COMPLETED = "completed"
    NO_SHOW = "no_show"
    DROPPED = "dropped"
    DECLINED = "declined"
    REGISTERED = "registered"
    IN_PROGRESS = "in_progress"


@dataclass(frozen=True)
class CareerGoal:
    role: str
    grade: Grade


@dataclass(frozen=True)
class Employee:
    employee_id: str
    role: str
    grade: Grade
    skills: dict[str, float]
    career_goal: CareerGoal | None = None


@dataclass(frozen=True)
class Requirement:
    role: str
    grade: Grade
    level: float
    critical: bool = False


@dataclass(frozen=True)
class Skill:
    skill_id: str
    name: str
    kind: SkillKind
    requirements: tuple[Requirement, ...]


@dataclass(frozen=True)
class SkillGain:
    skill_id: str
    gain: float
    max_level: float


@dataclass(frozen=True)
class Event:
    event_id: str
    title: str
    roles: tuple[str, ...]
    grades: tuple[Grade, ...]
    mandatory: bool
    self_paced: bool
    sessions: tuple[date, ...]
    prerequisites: tuple[str, ...]
    developed_skills: tuple[SkillGain, ...]
    mentoring: bool = False


@dataclass(frozen=True)
class Activity:
    employee_id: str
    event_id: str
    status: Status
    occurred_on: date
    completion_id: str | None = None


@dataclass(frozen=True)
class SkillGap:
    skill_id: str
    current_level: float
    required_level: float
    gap: float
    critical: bool


@dataclass(frozen=True)
class SkillEvidence:
    skill_id: str
    current_level: float
    required_level: float
    gap: float
    gain: float
    critical: bool


@dataclass(frozen=True)
class Evidence:
    cutoff_date: str
    target_role: str
    target_grade: str
    critical_gap: bool
    penalty: float
    fallback: bool
    skills: tuple[SkillEvidence, ...]
    fallback_reasons: tuple[str, ...] = ()


@dataclass(frozen=True)
class Recommendation:
    event_id: str
    title: str
    priority: float
    evidence: Evidence


@dataclass(frozen=True)
class RecommendationResponse:
    employee_id: str
    current_level: str
    target_level: str
    skill_gaps: list[SkillGap]
    recommendations: list[Recommendation]

    def to_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class AIExplanation:
    summary: str
    why: str
    next_steps: list[str]

    def to_dict(self) -> dict[str, object]:
        return asdict(self)
