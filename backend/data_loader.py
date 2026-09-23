"""Load the hackathon dataset without assuming one JSON envelope shape."""

from __future__ import annotations

import csv
import json
from dataclasses import dataclass, field
from datetime import date
from pathlib import Path
from typing import Any

from .models import Event, EventSkillGain


class DataLoadError(RuntimeError):
    """Raised when a required dataset file is missing or malformed."""


@dataclass(frozen=True)
class DataBundle:
    employees: list[dict[str, Any]]
    events: list[Event]
    skills: list[dict[str, Any]]
    activity_history: list[dict[str, Any]]
    role_profiles: list[dict[str, Any]] = field(default_factory=list)


def load_all_data(data_dir: str | Path = "data") -> DataBundle:
    """Load employees, events, skills and activity history from ``data_dir``.

    JSON files may be either arrays or objects containing a conventional
    collection key. CSV headers are preserved as strings for traceability.
    """

    root = Path(data_dir)
    skills_path = root / "skills.json"
    try:
        skill_document = json.loads(skills_path.read_text(encoding="utf-8-sig"))
        profiles = skill_document["role_profiles"]
        if not isinstance(profiles, list) or not all(isinstance(p, dict) for p in profiles):
            raise ValueError("role_profiles must be an array of objects")
        events = [_parse_event(row) for row in _load_collection(root / "events.json", ("events",))]
        event_ids = {e.event_id for e in events}
        if len(event_ids) != len(events):
            raise ValueError("duplicate event_id")
        for event in events:
            if any(p not in event_ids or p == event.event_id for p in event.prerequisite_events):
                raise ValueError("invalid prerequisite_events reference")
    except (OSError, UnicodeError, ValueError, KeyError, TypeError) as exc:
        raise DataLoadError(f"Invalid official dataset: {exc}") from exc
    result = DataBundle(
        employees=_load_collection(root / "employees.json", ("employees", "data", "items")),
        events=events,
        skills=_load_collection(root / "skills.json", ("skills", "data", "items")),
        activity_history=_load_csv(root / "activity_history.csv"),
        role_profiles=profiles,
    )
    skill_ids = {s["skill_id"] for s in result.skills}
    valid_roles = {p["role"] for p in profiles}
    for event in events:
        if not set(event.target_roles).issubset(valid_roles):
            raise DataLoadError(f"{event.event_id}: unknown target role")
        if not (set(event.prerequisites) | {g.skill_id for g in event.develops_skills}).issubset(
            skill_ids
        ):
            raise DataLoadError(f"{event.event_id}: unknown skill")
    return result


def _parse_event(row: dict[str, Any]) -> Event:
    """Accept the actual event schema; do not silently guess legacy aliases."""

    def text(key: str) -> str:
        value = row[key]
        if not isinstance(value, str) or not value.strip():
            raise ValueError(f"{key} must be a nonempty string")
        return value

    def strings(key: str) -> tuple[str, ...]:
        values = row[key]
        if not isinstance(values, list) or not all(isinstance(v, str) and v for v in values):
            raise ValueError(f"{key} must be a string array")
        return tuple(values)

    if not isinstance(row["mandatory"], bool):
        raise ValueError("mandatory must be boolean")
    grades = strings("target_grades")
    if not set(grades).issubset({"Junior", "Middle", "Senior", "Lead"}):
        raise ValueError("unknown target grade")
    gains = row["develops_skills"]
    if not isinstance(gains, list):
        raise ValueError("develops_skills must be an array")
    parsed: list[EventSkillGain] = []
    for gain in gains:
        if not isinstance(gain, dict) or not isinstance(gain.get("skill_id"), str):
            raise ValueError("develops_skills entries must contain skill_id")
        if type(gain.get("gain")) is not int or gain["gain"] <= 0:
            raise ValueError("gain must be a positive integer")
        if type(gain.get("max_level")) is not int or not 1 <= gain["max_level"] <= 5:
            raise ValueError("max_level must be between 1 and 5")
        parsed.append(EventSkillGain(gain["skill_id"], gain["gain"], gain["max_level"]))
    if len({g.skill_id for g in parsed}) != len(parsed):
        raise ValueError("duplicate developed skill")
    prerequisites = row["prerequisites"]
    if not isinstance(prerequisites, dict) or not all(
        isinstance(k, str) and type(v) is int and 0 <= v <= 5 for k, v in prerequisites.items()
    ):
        raise ValueError("prerequisites must map skill IDs to levels")
    event_format = text("format")
    if event_format not in {"online", "offline", "self_paced"}:
        raise ValueError("unknown format")
    sessions = tuple(date.fromisoformat(value) for value in strings("upcoming_sessions"))
    if (event_format == "self_paced" and sessions) or (
        event_format != "self_paced" and not sessions
    ):
        raise ValueError("upcoming_sessions does not match format")
    duration = row["duration_hours"]
    if type(duration) not in (int, float) or not 0 < duration < float("inf"):
        raise ValueError("duration_hours must be finite and positive")
    return Event(
        text("event_id"),
        text("title"),
        text("description"),
        text("type"),
        event_format,
        float(duration),
        row["mandatory"],
        strings("target_roles"),
        grades,
        tuple(parsed),
        dict(prerequisites),
        sessions,
        strings("prerequisite_events") if "prerequisite_events" in row else (),
    )


def _load_collection(path: Path, keys: tuple[str, ...]) -> list[dict[str, Any]]:
    if not path.is_file():
        raise DataLoadError(f"Required dataset file is missing: {path}")
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise DataLoadError(f"Could not read JSON dataset {path}: {exc}") from exc
    if isinstance(payload, list):
        rows = payload
    elif isinstance(payload, dict):
        rows = next((payload[key] for key in keys if isinstance(payload.get(key), list)), None)
        if rows is None:
            raise DataLoadError(f"JSON dataset {path} must contain a list")
    else:
        raise DataLoadError(f"JSON dataset {path} must contain an object or list")
    if not all(isinstance(row, dict) for row in rows):
        raise DataLoadError(f"JSON dataset {path} contains a non-object row")
    return rows


def _load_csv(path: Path) -> list[dict[str, Any]]:
    if not path.is_file():
        raise DataLoadError(f"Required dataset file is missing: {path}")
    try:
        with path.open(newline="", encoding="utf-8-sig") as source:
            return [dict(row) for row in csv.DictReader(source)]
    except (OSError, UnicodeError, csv.Error) as exc:
        raise DataLoadError(f"Could not read CSV dataset {path}: {exc}") from exc
