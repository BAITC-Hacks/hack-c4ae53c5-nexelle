"""Localized templates and a bounded asynchronous explanation adapter."""

from __future__ import annotations

import asyncio
from typing import Protocol

from .models import AIExplanation, Locale, Recommendation, RecommendationResponse


class ExplanationProvider(Protocol):
    async def generate(
        self,
        recommendation: Recommendation,
        *,
        locale: Locale,
    ) -> AIExplanation:
        """Use nonblocking I/O, honor cancellation and return text only."""
        ...


def explain(
    response: RecommendationResponse,
    recommendation: Recommendation | None = None,
    *,
    locale: Locale = Locale.RU,
) -> AIExplanation:
    locale = Locale(locale)
    selected = recommendation or (response.recommendations[0] if response.recommendations else None)
    if selected is None:
        return AIExplanation(
            "Нет доступных мероприятий." if locale == Locale.RU else "Қолжетімді іс-шаралар жоқ.",
            "Проверьте расписание и условия участия."
            if locale == Locale.RU
            else "Кесте мен қатысу шарттарын тексеріңіз.",
            [],
        )
    evidence = selected.evidence
    coverage = sum(s.gain for s in evidence.skills)
    if locale == Locale.KK:
        why = (
            "Мақсатты деңгейге қажетті дағдылар игерілген. Ұсыныс дағдыларды қолдауға және дамытуға арналған."
            if evidence.fallback
            else f"Іс-шара дағды алшақтығын {coverage:g} ұпайға дейін азайтады."
        )
        if evidence.critical_gap:
            why += " Маңызды дағдылардағы алшақтыққа басымдық беріледі."
        if evidence.penalty:
            why += f" Қатысу тарихы бойынша шегерім: {evidence.penalty:g}."
        return AIExplanation(
            f"Келесі қадам: {selected.title}.",
            why,
            [
                "Қатысу шарттарын тексеріңіз.",
                "Іс-шараға қатысыңыз.",
                "Аяқталған соң прогресті жаңартыңыз.",
            ],
        )
    why = (
        "Требования целевого грейда выполнены. Рекомендация поддерживает навыки и дальнейшее развитие."
        if evidence.fallback
        else f"Мероприятие сокращает разрыв в навыках на величину до {coverage:g}."
    )
    if evidence.critical_gap:
        why += " Критический разрыв имеет приоритет независимо от штрафов."
    if evidence.penalty:
        why += f" Штраф по истории участия: {evidence.penalty:g}."
    return AIExplanation(
        f"Следующий шаг: {selected.title}.",
        why,
        [
            "Проверьте условия участия.",
            "Пройдите мероприятие.",
            "После завершения обновите прогресс.",
        ],
    )


class ExplanationAdapter:
    def __init__(
        self,
        provider: ExplanationProvider | None = None,
        *,
        timeout_seconds: float = 1.4,
    ) -> None:
        if not 0 < timeout_seconds <= 1.4:
            raise ValueError("timeout_seconds must be in (0, 1.4] to reserve template time")
        self.provider = provider
        self.timeout = timeout_seconds
        self._pending: set[asyncio.Task[AIExplanation]] = set()

    async def explain(
        self,
        response: RecommendationResponse,
        *,
        locale: Locale = Locale.RU,
    ) -> AIExplanation:
        locale = Locale(locale)
        fallback = explain(response, locale=locale)
        if self.provider is None or not response.recommendations:
            return fallback
        # At most one outstanding request per adapter, including cancellation-resistant providers.
        if self._pending:
            return fallback
        task = asyncio.create_task(
            self.provider.generate(response.recommendations[0], locale=locale)
        )
        self._pending.add(task)
        task.add_done_callback(self._finished)
        try:
            done, _ = await asyncio.wait({task}, timeout=self.timeout)
            if not done:
                task.cancel()
                return fallback
            result = task.result()
            if not _valid(result):
                return fallback
            return result
        except asyncio.CancelledError:
            task.cancel()
            raise
        except Exception:
            return fallback

    def _finished(self, task: asyncio.Task[AIExplanation]) -> None:
        self._pending.discard(task)
        if not task.cancelled():
            task.exception()


def _valid(result: AIExplanation) -> bool:
    return (
        isinstance(result, AIExplanation)
        and isinstance(result.summary, str)
        and bool(result.summary.strip())
        and isinstance(result.why, str)
        and bool(result.why.strip())
        and isinstance(result.next_steps, list)
        and all(isinstance(step, str) and bool(step.strip()) for step in result.next_steps)
    )
