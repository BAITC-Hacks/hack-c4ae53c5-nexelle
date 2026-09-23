"""Regression tests for the jury edge cases and validated import."""

from __future__ import annotations

import asyncio
import json
import tempfile
import time
import unittest
from dataclasses import replace
from datetime import date
from pathlib import Path

from backend.ai_agent import ExplanationAdapter, explain
from backend.data_loader import DataBundle, DataLoadError, load_all_data
from backend.models import (
    Activity,
    AIExplanation,
    CareerGoal,
    Employee,
    Event,
    Grade,
    Locale,
    Recommendation,
    Requirement,
    Skill,
    SkillGain,
    SkillKind,
    Status,
)
from backend.progress import complete_event, effective_gain
from backend.recommendation_engine import RecommendationEngine

CUTOFF = date(2026, 10, 1)
PAST = date(2026, 9, 1)
FUTURE = date(2026, 10, 2)


def event(event_id: str = "EV_001", skill_id: str = "system_design") -> Event:
    return Event(
        event_id,
        "Практикум",
        ("Engineer",),
        tuple(Grade),
        False,
        True,
        (),
        (),
        (SkillGain(skill_id, 1, 5),),
    )


def bundle() -> DataBundle:
    return DataBundle(
        (Employee("E_001", "Engineer", Grade.MIDDLE, {"system_design": 1, "public_speaking": 1}),),
        (event(), event("EV_002", "public_speaking")),
        (
            Skill(
                "system_design",
                "Проектирование систем",
                SkillKind.HARD,
                (
                    Requirement("Engineer", Grade.SENIOR, 3, True),
                    Requirement("Engineer", Grade.LEAD, 3, True),
                ),
            ),
            Skill(
                "public_speaking",
                "Публичные выступления",
                SkillKind.SOFT,
                (
                    Requirement("Engineer", Grade.SENIOR, 2),
                    Requirement("Engineer", Grade.LEAD, 2),
                ),
            ),
        ),
        (),
    )


class RecommendationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.data = bundle()

    def recommendations(self, data: DataBundle | None = None) -> list[Recommendation]:
        return (
            RecommendationEngine(data or self.data, absolute_max_skill_level=5)
            .recommend(
                "E_001",
                cutoff_date=CUTOFF,
            )
            .recommendations
        )

    def test_critical_gap_outranks_any_penalty(self) -> None:
        history = tuple(Activity("E_001", "EV_001", Status.NO_SHOW, PAST) for _ in range(100))
        result = self.recommendations(replace(self.data, activity_history=history))
        self.assertEqual(result[0].event_id, "EV_001")
        self.assertLess(result[0].priority, result[1].priority)
        self.assertEqual(result[0].evidence.penalty, 1500)

    def test_all_negative_statuses_penalize(self) -> None:
        for status, penalty in ((Status.NO_SHOW, 15), (Status.DROPPED, 10), (Status.DECLINED, 5)):
            with self.subTest(status=status):
                data = replace(
                    self.data, activity_history=(Activity("E_001", "EV_001", status, PAST),)
                )
                self.assertEqual(self.recommendations(data)[0].evidence.penalty, penalty)

    def test_filters(self) -> None:
        rejected = (
            replace(event(), mandatory=True),
            replace(event(), roles=("Designer",)),
            replace(event(), grades=(Grade.JUNIOR,)),
            replace(event(), prerequisites=("EV_002",)),
            replace(event(), self_paced=False, sessions=()),
            replace(event(), self_paced=False, sessions=(CUTOFF,)),
            replace(event(), self_paced=False, sessions=(PAST,)),
            replace(event(), developed_skills=(SkillGain("system_design", 1, 1),)),
            replace(event(), developed_skills=()),
        )
        for candidate in rejected:
            with self.subTest(candidate=candidate):
                self.assertEqual(self.recommendations(replace(self.data, events=(candidate,))), [])

    def test_future_session(self) -> None:
        data = replace(
            self.data, events=(replace(event(), self_paced=False, sessions=(PAST, FUTURE)),)
        )
        self.assertEqual(len(self.recommendations(data)), 1)

    def test_completed_excluded_except_club(self) -> None:
        data = replace(
            self.data,
            events=(event(), event("EV_036")),
            activity_history=(
                Activity("E_001", "EV_001", Status.COMPLETED, PAST),
                Activity("E_001", "EV_036", Status.COMPLETED, PAST),
            ),
        )
        self.assertEqual([r.event_id for r in self.recommendations(data)], ["EV_036"])

    def test_prerequisites_require_completed_before_cutoff(self) -> None:
        for status, occurred_on, expected in (
            (Status.COMPLETED, PAST, 1),
            (Status.NO_SHOW, PAST, 0),
            (Status.COMPLETED, FUTURE, 0),
        ):
            with self.subTest(status=status, occurred_on=occurred_on):
                data = replace(
                    self.data,
                    events=(replace(event(), prerequisites=("EV_002",)),),
                    activity_history=(Activity("E_001", "EV_002", status, occurred_on),),
                )
                self.assertEqual(len(self.recommendations(data)), expected)

    def test_future_history_is_ignored(self) -> None:
        data = replace(
            self.data, activity_history=(Activity("E_001", "EV_001", Status.COMPLETED, FUTURE),)
        )
        self.assertEqual(self.recommendations(data), self.recommendations())

    def test_other_employee_history_is_ignored(self) -> None:
        data = replace(
            self.data, activity_history=(Activity("E_002", "EV_001", Status.COMPLETED, PAST),)
        )
        self.assertEqual(self.recommendations(data), self.recommendations())

    def test_explicit_goal_matches_role_and_grade_together(self) -> None:
        employee = replace(self.data.employees[0], career_goal=CareerGoal("Designer", Grade.SENIOR))
        skills = tuple(
            replace(s, requirements=(*s.requirements, Requirement("Designer", Grade.SENIOR, 3)))
            for s in self.data.skills
        )
        for role, grade, expected in (
            ("Designer", Grade.SENIOR, 1),
            ("Designer", Grade.MIDDLE, 0),
            ("Engineer", Grade.MIDDLE, 1),
        ):
            data = replace(
                self.data,
                employees=(employee,),
                skills=skills,
                events=(replace(event(), roles=(role,), grades=(grade,)),),
            )
            with self.subTest(role=role, grade=grade):
                self.assertEqual(len(self.recommendations(data)), expected)

    def test_lead_and_zero_gap_fallback(self) -> None:
        for grade in (Grade.MIDDLE, Grade.LEAD):
            employee = replace(
                self.data.employees[0],
                grade=grade,
                skills={"system_design": 5, "public_speaking": 5},
            )
            mentoring = replace(event("EV_003"), developed_skills=(), mentoring=True)
            data = replace(self.data, employees=(employee,), events=(*self.data.events, mentoring))
            result = self.recommendations(data)
            self.assertEqual(len(result), 3)
            self.assertTrue(all(r.evidence.fallback for r in result))

    def test_fallback_preserves_eligibility(self) -> None:
        employee = replace(
            self.data.employees[0], skills={"system_design": 5, "public_speaking": 5}
        )
        data = replace(self.data, employees=(employee,), events=(replace(event(), mandatory=True),))
        self.assertEqual(self.recommendations(data), [])

    def test_missing_requirements_is_not_zero_gap(self) -> None:
        with self.assertRaises(ValueError):
            self.recommendations(replace(self.data, skills=()))

    def test_caps_and_multiple_skills_in_evidence(self) -> None:
        candidate = replace(
            event(),
            developed_skills=(
                SkillGain("system_design", 10, 2),
                SkillGain("public_speaking", 10, 5),
            ),
        )
        result = self.recommendations(replace(self.data, events=(candidate,)))[0]
        self.assertEqual([e.gain for e in result.evidence.skills], [1, 1])

    def test_deterministic_tie_break(self) -> None:
        data = replace(self.data, events=(event("EV_B"), event("EV_A")))
        self.assertEqual([r.event_id for r in self.recommendations(data)], ["EV_A", "EV_B"])
        self.assertEqual(
            self.recommendations(data),
            self.recommendations(replace(data, events=tuple(reversed(data.events)))),
        )

    def test_explicit_cutoff_and_valid_limit(self) -> None:
        engine = RecommendationEngine(self.data, absolute_max_skill_level=5)
        with self.assertRaises(TypeError):
            engine.recommend("E_001")  # type: ignore[call-arg]
        for limit in (0, 4, True, 1.5):
            with self.assertRaises(ValueError):
                engine.recommend("E_001", cutoff_date=CUTOFF, limit=limit)  # type: ignore[arg-type]


class ProgressTests(unittest.TestCase):
    def test_formula(self) -> None:
        for current, gain, cap, maximum, expected in (
            (1, 2, 4, 5, 2),
            (3, 3, 4, 5, 1),
            (3, 10, 10, 5, 2),
            (4, 3, 4, 5, 0),
            (5, 3, 4, 5, 0),
        ):
            with self.subTest(current=current, cap=cap):
                self.assertEqual(
                    effective_gain(current, SkillGain("s", gain, cap), maximum), expected
                )

    def test_invalid_levels(self) -> None:
        for gain in (-1, float("nan"), float("inf")):
            with self.assertRaises(ValueError):
                effective_gain(1, SkillGain("s", gain, 5), 5)

    def test_completion_and_idempotency(self) -> None:
        employee = bundle().employees[0]
        result = complete_event(
            employee,
            event(),
            (),
            completed_on=CUTOFF,
            completion_id="c1",
            absolute_max_skill_level=5,
        )
        self.assertEqual(result.employee.skills["system_design"], 2)
        self.assertEqual(employee.skills["system_design"], 1)
        retry = complete_event(
            result.employee,
            event(),
            result.history,
            completed_on=CUTOFF,
            completion_id="c1",
            absolute_max_skill_level=5,
        )
        self.assertFalse(retry.applied)
        self.assertEqual(retry.history, result.history)
        with self.assertRaises(ValueError):
            complete_event(
                result.employee,
                event("other"),
                result.history,
                completed_on=CUTOFF,
                completion_id="c1",
                absolute_max_skill_level=5,
            )

    def test_event_without_skills_recorded(self) -> None:
        employee = bundle().employees[0]
        result = complete_event(
            employee,
            replace(event(), developed_skills=()),
            (),
            completed_on=CUTOFF,
            completion_id="c1",
            absolute_max_skill_level=5,
        )
        self.assertEqual(result.employee, employee)
        self.assertEqual(len(result.history), 1)

    def test_club_repeat(self) -> None:
        first = complete_event(
            bundle().employees[0],
            event("EV_036"),
            (),
            completed_on=PAST,
            completion_id="c1",
            absolute_max_skill_level=5,
        )
        second = complete_event(
            first.employee,
            event("EV_036"),
            first.history,
            completed_on=CUTOFF,
            completion_id="c2",
            absolute_max_skill_level=5,
        )
        self.assertEqual(second.employee.skills["system_design"], 3)
        self.assertEqual(len(second.history), 2)


class ImportTests(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.write(
            "employees",
            [{"employee_id": "E_001", "role": "Engineer", "grade": "Middle", "skills": {"s": 1}}],
        )
        self.write(
            "skills",
            [
                {
                    "skill_id": "s",
                    "name": "Навык",
                    "kind": "hard",
                    "requirements": [{"role": "Engineer", "grade": "Senior", "level": 3}],
                }
            ],
        )
        self.events = [
            {
                "event_id": "EV_001",
                "title": "Курс",
                "roles": ["Engineer"],
                "grades": ["Middle"],
                "mandatory": False,
                "self_paced": True,
                "sessions": [],
                "prerequisites": [],
                "developed_skills": [{"skill_id": "s", "gain": 1, "max_level": 5}],
            }
        ]
        self.write("events", self.events)
        (self.root / "activity_history.csv").write_text(
            "employee_id,event_id,status,occurred_on\n", encoding="utf-8"
        )

    def write(self, name: str, rows: object) -> None:
        (self.root / f"{name}.json").write_text(json.dumps(rows), encoding="utf-8")

    def test_typed_import(self) -> None:
        data = load_all_data(self.root)
        self.assertEqual(data.employees[0].grade, Grade.MIDDLE)
        self.assertEqual(
            len(
                RecommendationEngine(data, absolute_max_skill_level=5)
                .recommend("E_001", cutoff_date=CUTOFF)
                .recommendations
            ),
            1,
        )

    def test_rejects_invalid_types_and_references(self) -> None:
        for key, value in (
            ("mandatory", "false"),
            ("prerequisites", ["missing"]),
            ("event_id", ""),
        ):
            with self.subTest(key=key):
                self.write("events", [{**self.events[0], key: value}])
                with self.assertRaises(DataLoadError):
                    load_all_data(self.root)

    def test_duplicate_event(self) -> None:
        self.write("events", self.events * 2)
        with self.assertRaises(DataLoadError):
            load_all_data(self.root)

    def test_missing_file(self) -> None:
        with self.assertRaises(DataLoadError):
            load_all_data(self.root / "missing")


class ExplanationTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.response = RecommendationEngine(bundle(), absolute_max_skill_level=5).recommend(
            "E_001", cutoff_date=CUTOFF
        )

    async def test_templates_both_locales(self) -> None:
        ru = await ExplanationAdapter().explain(self.response, locale=Locale.RU)
        kk = await ExplanationAdapter().explain(self.response, locale=Locale.KK)
        self.assertNotEqual(ru.why, kk.why)
        self.assertEqual(len(ru.next_steps), 3)

    async def test_timeout(self) -> None:
        class SlowProvider:
            async def generate(
                self, recommendation: Recommendation, *, locale: Locale
            ) -> AIExplanation:
                await asyncio.sleep(10)
                return AIExplanation("late", "late", [])

        start = time.monotonic()
        actual = await ExplanationAdapter(SlowProvider(), timeout_seconds=0.02).explain(
            self.response
        )
        self.assertLess(time.monotonic() - start, 0.5)
        self.assertEqual(actual, explain(self.response))

    async def test_provider_failure(self) -> None:
        class BrokenProvider:
            async def generate(
                self, recommendation: Recommendation, *, locale: Locale
            ) -> AIExplanation:
                raise OSError("offline")

        self.assertEqual(
            await ExplanationAdapter(BrokenProvider()).explain(self.response),
            explain(self.response),
        )

    async def test_provider_success_preserves_evidence(self) -> None:
        class Provider:
            async def generate(
                self, recommendation: Recommendation, *, locale: Locale
            ) -> AIExplanation:
                return AIExplanation("Текст", "Объяснение", [])

        before = self.response.to_dict()
        actual = await ExplanationAdapter(Provider()).explain(self.response)
        self.assertEqual(actual.summary, "Текст")
        self.assertEqual(before, self.response.to_dict())


if __name__ == "__main__":
    unittest.main()
