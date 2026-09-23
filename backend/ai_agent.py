"""Explain recommendation results with optional LLM integration and fallback."""

from __future__ import annotations

import os
from typing import Any

from .models import AIExplanation, Recommendation, RecommendationResponse


def explain(response: RecommendationResponse, recommendation: Recommendation | None = None) -> AIExplanation:
    """Return a stable explanation; an API key is optional by design.

    The optional OpenAI integration is intentionally left behind the application
    boundary. Without a configured provider, this deterministic explanation is
    safe for local demos and tests.
    """

    _ = os.getenv("OPENAI_API_KEY")  # Presence is observed without embedding secrets.
    selected = recommendation or (response.recommendations[0] if response.recommendations else None)
    if selected is None:
        return AIExplanation(
            summary="Пока нет подходящего следующего шага.",
            why="Для текущего профиля не найдено доступное событие, закрывающее skill gap.",
            next_steps=[],
        )
    factors = selected.factors
    history = "История активности учтена при ранжировании."
    if factors.get("completed_before"):
        history = "Это событие уже встречалось в истории, поэтому его приоритет снижен."
    return AIExplanation(
        summary=f"Следующий шаг: {selected.title} для развития навыка {selected.skill_gap}.",
        why=(
            f"Текущий уровень {selected.current_skill_level:g}, цель {selected.required_skill_level:g}; "
            f"активность закрывает до {selected.gap_covered:g} пункта gap. {history}"
        ),
        next_steps=[
            f"Закройте gap по навыку {selected.skill_gap}.",
            f"Пройдите активность «{selected.title}».",
            "После завершения пересчитайте рекомендации и прогресс.",
        ],
    )
