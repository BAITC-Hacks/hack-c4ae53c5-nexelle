"""Command-line entry point using the same contracts as future API handlers."""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys
from datetime import date

from .ai_agent import ExplanationAdapter
from .data_loader import load_all_data
from .models import Locale
from .recommendation_engine import RecommendationEngine


def main() -> int:
    parser = argparse.ArgumentParser(description="Career Quest: рекомендации и объяснения")
    parser.add_argument("--data-dir", default="data")
    parser.add_argument("--employee-id", required=True)
    parser.add_argument("--cutoff-date", required=True, type=date.fromisoformat)
    parser.add_argument(
        "--absolute-max-skill-level", type=float, default=os.getenv("ABSOLUTE_MAX_SKILL_LEVEL")
    )
    parser.add_argument("--locale", choices=[locale.value for locale in Locale], default="ru")
    args = parser.parse_args()
    if args.absolute_max_skill_level is None:
        parser.error("Set --absolute-max-skill-level or ABSOLUTE_MAX_SKILL_LEVEL")
    try:
        engine = RecommendationEngine(
            load_all_data(args.data_dir),
            absolute_max_skill_level=args.absolute_max_skill_level,
        )
        response = engine.recommend(args.employee_id, cutoff_date=args.cutoff_date)
        explanation = asyncio.run(
            ExplanationAdapter().explain(response, locale=Locale(args.locale))
        )
        output = response.to_dict()
        output["explanation"] = explanation.to_dict()
        print(json.dumps(output, ensure_ascii=False, allow_nan=False, indent=2))
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
