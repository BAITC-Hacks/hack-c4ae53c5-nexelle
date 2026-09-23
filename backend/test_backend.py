"""Regression tests using the unmodified official files and schema."""

from __future__ import annotations

import copy
import json
import tempfile
import unittest
from dataclasses import replace
from datetime import date
from pathlib import Path

from backend.ai_agent import explain
from backend.data_loader import DataLoadError, load_all_data
from backend.recommendation_engine import RecommendationEngine, effective_gain

DATA = Path(__file__).resolve().parents[1] / "data"
CUTOFF = date(2026, 10, 1)


class OfficialDatasetTests(unittest.TestCase):
    def test_load_official_events_and_all_employees(self) -> None:
        data = load_all_data(DATA)
        self.assertEqual(
            (len(data.employees), len(data.events), len(data.skills), len(data.role_profiles)),
            (200, 40, 60, 32),
        )
        self.assertEqual(len(data.activity_history), 2743)
        self.assertEqual(sum(e.mandatory for e in data.events), 4)
        self.assertEqual(sum(bool(e.prerequisites) for e in data.events), 12)
        self.assertEqual(sum(e.format == "self_paced" for e in data.events), 9)
        engine = RecommendationEngine(data)
        recommendations = 0
        for employee in data.employees:
            result = engine.recommend(employee["employee_id"], cutoff_date=CUTOFF)
            self.assertLessEqual(len(result.recommendations), 3)
            self.assertTrue(explain(result).summary)
            recommendations += len(result.recommendations)
        self.assertGreater(recommendations, 0)

    def test_legacy_event_fields_rejected(self) -> None:
        for canonical, legacy in (
            ("target_roles", "roles"),
            ("target_grades", "grades"),
            ("develops_skills", "developed_skills"),
        ):
            with self.subTest(field=canonical), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                for source in DATA.glob("*.json"):
                    (root / source.name).write_bytes(source.read_bytes())
                (root / "activity_history.csv").write_bytes(
                    (DATA / "activity_history.csv").read_bytes()
                )
                document = json.loads((root / "events.json").read_text(encoding="utf-8"))
                document["events"][0][legacy] = document["events"][0].pop(canonical)
                (root / "events.json").write_text(json.dumps(document), encoding="utf-8")
                with self.assertRaises(DataLoadError):
                    load_all_data(root)

    def test_actual_gain_caps(self) -> None:
        self.assertEqual(effective_gain(5, 1, 3), 0)
        self.assertEqual(effective_gain(4, 10, 10), 1)
        self.assertEqual(effective_gain(2, 2, 3), 1)

    def test_critical_system_design_with_negative_history(self) -> None:
        data = load_all_data(DATA)
        employee = copy.deepcopy(data.employees[0])
        employee.update(
            role="Backend Engineer",
            grade="Middle",
            career_goal={"target_role": "Backend Engineer", "target_grade": "Senior"},
            last_review_date="2026-09-30",
            skills={"SK_SYSTEM_DESIGN": 1, "SK_PUBLIC_SPEAKING": 1},
        )
        profile = {
            "role": "Backend Engineer",
            "grade": "Senior",
            "required_skills": {"SK_SYSTEM_DESIGN": 4, "SK_PUBLIC_SPEAKING": 2},
            "critical_skills": ["SK_SYSTEM_DESIGN"],
        }
        system = next(
            e
            for e in data.events
            if any(g.skill_id == "SK_SYSTEM_DESIGN" for g in e.develops_skills)
        )
        public = next(
            e
            for e in data.events
            if any(g.skill_id == "SK_PUBLIC_SPEAKING" for g in e.develops_skills)
        )
        system = replace(
            system,
            format="online",
            prerequisites={},
            target_roles=("Backend Engineer",),
            target_grades=("Middle",),
            upcoming_sessions=(date(2026, 10, 2),),
            develops_skills=tuple(
                g for g in system.develops_skills if g.skill_id == "SK_SYSTEM_DESIGN"
            ),
        )
        public = replace(
            public,
            format="offline",
            prerequisites={},
            target_roles=("Backend Engineer",),
            target_grades=("Middle",),
            upcoming_sessions=(date(2026, 10, 2),),
            develops_skills=tuple(
                g for g in public.develops_skills if g.skill_id == "SK_PUBLIC_SPEAKING"
            ),
        )
        history = [
            {
                "record_id": str(i),
                "employee_id": employee["employee_id"],
                "event_id": system.event_id,
                "status": status,
                "date": "2026-09-20",
            }
            for i, status in enumerate(["no_show", "dropped", "declined"] * 10)
        ]
        custom = replace(
            data,
            employees=[employee],
            events=[system, public],
            role_profiles=[profile],
            activity_history=history,
        )
        result = RecommendationEngine(custom).recommend(employee["employee_id"], cutoff_date=CUTOFF)
        self.assertEqual(result.recommendations[0].event_id, system.event_id)
        self.assertLess(result.recommendations[0].priority, result.recommendations[1].priority)

    def test_zero_gap_lead_fallback(self) -> None:
        data = load_all_data(DATA)
        employee = copy.deepcopy(data.employees[0])
        employee.update(
            grade="Lead",
            career_goal=None,
            last_review_date="2026-09-30",
            skills={s["skill_id"]: 5 for s in data.skills},
        )
        custom = replace(data, employees=[employee], activity_history=[])
        result = RecommendationEngine(custom).recommend(employee["employee_id"], cutoff_date=CUTOFF)
        self.assertFalse(result.skill_gaps)
        self.assertGreaterEqual(len(result.recommendations), 1)
        self.assertTrue(all(r.factors["fallback"] for r in result.recommendations))

    def test_cutoff_required_and_stable(self) -> None:
        engine = RecommendationEngine(load_all_data(DATA))
        with self.assertRaises(TypeError):
            engine.recommend("E0001")  # type: ignore[call-arg]
        self.assertEqual(
            engine.recommend("E0001", cutoff_date=CUTOFF).to_dict(),
            engine.recommend("E0001", cutoff_date=CUTOFF).to_dict(),
        )


if __name__ == "__main__":
    unittest.main()
