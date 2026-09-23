# Career Quest

AI-зона проекта анализирует профиль сотрудника и историю активностей, считает пробелы в навыках и возвращает до трёх детерминированных рекомендаций с объяснением.

## Структура

- `backend/data_loader.py` — загрузка `data/employees.json`, `events.json`, `skills.json` и `activity_history.csv`.
- `backend/recommendation_engine.py` — расчёт skill gaps и ранжирование событий.
- `backend/ai_agent.py` — структурированное объяснение с детерминированным fallback без API key.
- `backend/models.py` — модели результата на стандартной библиотеке Python.
- `backend/test_backend.py` — smoke-тесты загрузки, ranking и fallback.

## Запуск тестов

```sh
python -m unittest backend.test_backend -v
```

Реальные файлы dataset должны находиться в `data/`. Текущая ветка содержит только код загрузки и анализа: production dataset в рабочем дереве отсутствует, поэтому он не создаётся искусственно.

## Контракт результата

`RecommendationEngine(data).recommend(employee_id)` возвращает `RecommendationResponse` с employee id, текущим и целевым уровнем, `skill_gaps` и отсортированными рекомендациями. Одинаковые входные данные дают одинаковый порядок и score. Уже завершённые активности получают пониженный приоритет.
