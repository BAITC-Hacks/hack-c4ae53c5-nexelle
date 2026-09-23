"""Validated JSON/CSV boundary; the engine receives typed domain objects."""

from __future__ import annotations

import csv
import json
import math
from dataclasses import dataclass
from datetime import date
from pathlib import Path

from .models import (
    Activity,
    CareerGoal,
    Employee,
    Event,
    Grade,
    Requirement,
    Skill,
    SkillGain,
    SkillKind,
    Status,
)


class DataLoadError(ValueError):
    """A missing file, invalid field or inconsistent dataset reference."""


@dataclass(frozen=True)
class DataBundle:
    employees: tuple[Employee, ...]
    events: tuple[Event, ...]
    skills: tuple[Skill, ...]
    activity_history: tuple[Activity, ...]


def load_all_data(data_dir: str | Path = "data") -> DataBundle:
    root = Path(data_dir)
    try:
        employees = tuple(_employee(r) for r in _collection(root / "employees.json", "employees"))
        events = tuple(_event(r) for r in _collection(root / "events.json", "events"))
        skills = tuple(_skill(r) for r in _collection(root / "skills.json", "skills"))
        with (root / "activity_history.csv").open(newline="", encoding="utf-8-sig") as source:
            reader = csv.DictReader(source, strict=True)
            required = {"employee_id", "event_id", "status", "occurred_on"}
            if reader.fieldnames is None or not required.issubset(reader.fieldnames):
                raise DataLoadError("activity_history.csv: missing required columns")
            history = tuple(_activity(dict(row)) for row in reader)
        bundle = DataBundle(employees, events, skills, history)
        _validate_references(bundle)
        return bundle
    except (OSError, UnicodeError, ValueError, KeyError, TypeError, csv.Error) as exc:
        raise DataLoadError(f"Invalid dataset in {root}: {exc}") from exc


def _object(value: object) -> dict[str, object]:
    if not isinstance(value, dict) or not all(isinstance(k, str) for k in value):
        raise DataLoadError("Expected an object with string keys")
    return value


def _array(value: object) -> list[object]:
    if not isinstance(value, list):
        raise DataLoadError("Expected an array")
    return value


def _text(value: object) -> str:
    if not isinstance(value, str) or not value.strip():
        raise DataLoadError("Expected a nonempty string")
    return value


def _number(value: object) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise DataLoadError("Expected a number")
    if not math.isfinite(value) or value < 0:
        raise DataLoadError("Expected a finite nonnegative number")
    return float(value)


def _flag(row: dict[str, object], key: str, *, default: bool | None = None) -> bool:
    value = row.get(key, default)
    if not isinstance(value, bool):
        raise DataLoadError(f"{key}: expected boolean")
    return value


def _strings(value: object) -> tuple[str, ...]:
    result = tuple(_text(v) for v in _array(value))
    if len(set(result)) != len(result):
        raise DataLoadError("Duplicate array values")
    return result


def _collection(path: Path, key: str) -> list[dict[str, object]]:
    payload: object = json.loads(path.read_text(encoding="utf-8-sig"))
    if isinstance(payload, dict):
        payload = payload.get(key, payload.get("data", payload.get("items")))
    return [_object(r) for r in _array(payload)]


def _employee(row: dict[str, object]) -> Employee:
    goal = row.get("career_goal")
    career_goal = None
    if goal is not None:
        goal_row = _object(goal)
        career_goal = CareerGoal(_text(goal_row["role"]), Grade(_text(goal_row["grade"])))
    return Employee(
        _text(row["employee_id"]),
        _text(row["role"]),
        Grade(_text(row["grade"])),
        {k: _number(v) for k, v in _object(row["skills"]).items()},
        career_goal,
    )


def _event(row: dict[str, object]) -> Event:
    gains: list[SkillGain] = []
    for value in _array(row["developed_skills"]):
        gain = _object(value)
        gains.append(
            SkillGain(_text(gain["skill_id"]), _number(gain["gain"]), _number(gain["max_level"]))
        )
    if len({g.skill_id for g in gains}) != len(gains):
        raise DataLoadError("Duplicate developed skill")
    return Event(
        _text(row["event_id"]),
        _text(row["title"]),
        _strings(row["roles"]),
        tuple(Grade(v) for v in _strings(row["grades"])),
        _flag(row, "mandatory"),
        _flag(row, "self_paced"),
        tuple(date.fromisoformat(v) for v in _strings(row["sessions"])),
        _strings(row["prerequisites"]),
        tuple(gains),
        _flag(row, "mentoring", default=False),
    )


def _skill(row: dict[str, object]) -> Skill:
    requirements: list[Requirement] = []
    for value in _array(row["requirements"]):
        r = _object(value)
        requirements.append(
            Requirement(
                _text(r["role"]),
                Grade(_text(r["grade"])),
                _number(r["level"]),
                _flag(r, "critical", default=False),
            )
        )
    if len({(r.role, r.grade) for r in requirements}) != len(requirements):
        raise DataLoadError("Duplicate skill requirement for role/grade")
    return Skill(
        _text(row["skill_id"]),
        _text(row["name"]),
        SkillKind(_text(row["kind"])),
        tuple(requirements),
    )


def _activity(row: dict[str, object]) -> Activity:
    completion = row.get("completion_id")
    return Activity(
        _text(row["employee_id"]),
        _text(row["event_id"]),
        Status(_text(row["status"])),
        date.fromisoformat(_text(row["occurred_on"])),
        _text(completion) if completion else None,
    )


def _unique(values: list[str], kind: str) -> set[str]:
    if len(set(values)) != len(values):
        raise DataLoadError(f"Duplicate {kind} ID")
    if any(not v.isascii() or not v.strip() for v in values):
        raise DataLoadError(f"{kind} IDs must be nonempty ASCII identifiers")
    return set(values)


def _validate_references(data: DataBundle) -> None:
    employees = _unique([e.employee_id for e in data.employees], "employee")
    events = _unique([e.event_id for e in data.events], "event")
    skills = _unique([s.skill_id for s in data.skills], "skill")
    for employee in data.employees:
        if not set(employee.skills).issubset(skills):
            raise DataLoadError(f"{employee.employee_id}: unknown skill")
    for event in data.events:
        if not {g.skill_id for g in event.developed_skills}.issubset(skills):
            raise DataLoadError(f"{event.event_id}: unknown skill")
        if not set(event.prerequisites).issubset(events) or event.event_id in event.prerequisites:
            raise DataLoadError(f"{event.event_id}: invalid prerequisite")
    completion_ids: set[str] = set()
    for activity in data.activity_history:
        if activity.employee_id not in employees or activity.event_id not in events:
            raise DataLoadError("History references an unknown employee or event")
        if activity.completion_id:
            if activity.status != Status.COMPLETED or activity.completion_id in completion_ids:
                raise DataLoadError("Invalid or duplicate completion_id")
            completion_ids.add(activity.completion_id)
