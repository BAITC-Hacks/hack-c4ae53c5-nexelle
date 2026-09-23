"""Load the hackathon dataset without assuming one JSON envelope shape."""

from __future__ import annotations

import csv
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any


class DataLoadError(RuntimeError):
    """Raised when a required dataset file is missing or malformed."""


@dataclass(frozen=True)
class DataBundle:
    employees: list[dict[str, Any]]
    events: list[dict[str, Any]]
    skills: list[dict[str, Any]]
    activity_history: list[dict[str, Any]]


def load_all_data(data_dir: str | Path = "data") -> DataBundle:
    """Load employees, events, skills and activity history from ``data_dir``.

    JSON files may be either arrays or objects containing a conventional
    collection key. CSV headers are preserved as strings for traceability.
    """

    root = Path(data_dir)
    return DataBundle(
        employees=_load_collection(root / "employees.json", ("employees", "data", "items")),
        events=_load_collection(root / "events.json", ("events", "activities", "data", "items")),
        skills=_load_collection(root / "skills.json", ("skills", "data", "items")),
        activity_history=_load_csv(root / "activity_history.csv"),
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
