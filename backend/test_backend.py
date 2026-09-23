"""Smoke tests for data loading, deterministic ranking and AI fallback."""

from __future__ import annotations

import csv
import json
import tempfile
import unittest
from pathlib import Path

from backend.ai_agent import explain
from backend.data_loader import load_all_data
from backend.recommendation_engine import RecommendationEngine


class RecommendationEngineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.root = Path(tempfile.mkdtemp())
        (self.root / "employees.json").write_text(json.dumps([{"employee_id": "E-1", "role": "Designer", "grade": "Middle", "career_goal": {"grade": "Senior"}, "skills": {"S-1": 1}}]), encoding="utf-8")
        (self.root / "skills.json").write_text(json.dumps([{"skill_id": "S-1", "name": "Systems Thinking", "required_level": 3, "critical": True}]), encoding="utf-8")
        (self.root / "events.json").write_text(json.dumps([{"event_id": "EV-1", "title": "Systems lab", "skill_id": "S-1", "gain": 1, "availability": "available"}, {"event_id": "EV-2", "title": "Systems session", "skill_id": "S-1", "gain": 1, "availability": "waitlist"}]), encoding="utf-8")
        with (self.root / "activity_history.csv").open("w", newline="", encoding="utf-8") as output:
            writer = csv.DictWriter(output, fieldnames=["employee_id", "event_id", "status"])
            writer.writeheader()
            writer.writerow({"employee_id": "E-1", "event_id": "EV-2", "status": "completed"})

    def test_load_and_recommend(self) -> None:
        data = load_all_data(self.root)
        result = RecommendationEngine(data).recommend("E-1")
        self.assertEqual(len(result.skill_gaps), 1)
        self.assertEqual(len(result.recommendations), 2)
        self.assertEqual(result.recommendations[0].event_id, "EV-1")

    def test_ranking_is_deterministic(self) -> None:
        data = load_all_data(self.root)
        first = RecommendationEngine(data).recommend("E-1").to_dict()
        second = RecommendationEngine(data).recommend("E-1").to_dict()
        self.assertEqual(first, second)

    def test_ai_fallback(self) -> None:
        result = RecommendationEngine(load_all_data(self.root)).recommend("E-1")
        explanation = explain(result)
        self.assertTrue(explanation.summary)
        self.assertEqual(len(explanation.next_steps), 3)


if __name__ == "__main__":
    unittest.main()
