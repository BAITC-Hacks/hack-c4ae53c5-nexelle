# Контракт данных Career Quest

Контракт уточнён для строгой типизации. Старые произвольные aliases и извлечение чисел из строк больше не поддерживаются.
Перед импортом реального dataset его поля необходимо сопоставить с этим контрактом.
JSON — массив объектов либо объект с ключом коллекции (`employees`, `skills`, `events`), `data` или `items`.

## employees.json

```json
[
  {
    "employee_id": "E_001",
    "role": "Engineer",
    "grade": "Middle",
    "skills": {"system_design": 1},
    "career_goal": {"role": "Engineer", "grade": "Senior"}
  }
]
```

`career_goal` необязателен. Грейды: Junior, Middle, Senior, Lead.
Идентификаторы — ASCII; названия ролей и коды навыков должны совпадать точно.
Профиль содержит уже учтённые уровни навыков на дату среза. Импорт не начисляет историю повторно.
Для исторического среза нужен соответствующий снимок профиля; восстановление профиля из истории не реализовано.

## skills.json

```json
[
  {
    "skill_id": "system_design",
    "name": "Проектирование систем",
    "kind": "hard",
    "requirements": [
      {"role": "Engineer", "grade": "Senior", "level": 3, "critical": true}
    ]
  }
]
```

`kind`: hard или soft. Критичность задаётся для конкретной роли и грейда.
Дубли требований одной роли/грейда отклоняются. Если для целевой роли/грейда нет требований,
движок возвращает ошибку данных, поскольку отсутствие матрицы не означает нулевой gap.

## events.json

```json
[
  {
    "event_id": "EV_001",
    "title": "Практикум проектирования",
    "roles": ["Engineer"],
    "grades": ["Middle", "Senior"],
    "mandatory": false,
    "self_paced": false,
    "sessions": ["2026-10-02"],
    "prerequisites": [],
    "developed_skills": [
      {"skill_id": "system_design", "gain": 1, "max_level": 4}
    ],
    "mentoring": false
  }
]
```

Все поля, кроме `mentoring`, обязательны. Пустые roles/grades не означают доступность всем.
Для self-paced разрешён пустой список sessions. Разрешён пустой developed_skills.
Даты имеют формат YYYY-MM-DD. Сессия на саму дату среза уже не подходит.
Роль и грейд должны одновременно совпасть с текущим профилем либо с явной карьерной целью.
Все prerequisites должны ссылаться на известные мероприятия и иметь completed в истории данного сотрудника на дату среза.

## activity_history.csv

```csv
employee_id,event_id,status,occurred_on,completion_id
E_001,EV_001,no_show,2026-09-01,
```

Обязательные колонки: employee_id, event_id, status, occurred_on.
Статусы: completed, no_show, dropped, declined, registered, in_progress.
completion_id необязателен для исходной истории; если указан, он уникален и относится только к completed.
Новая операция завершения требует непустой completion_id.

Числа должны быть конечными и неотрицательными, флаги — JSON boolean.
Дубликаты ID и неизвестные ссылки отклоняются.
Размеры 200 сотрудников / 8 ролей / 60 навыков / 40 мероприятий описывают ожидаемый production dataset,
но не зашиты в универсальный импортёр.

## Python API

```python
from datetime import date
from backend.data_loader import load_all_data
from backend.recommendation_engine import RecommendationEngine

def recommendations(data_dir: str, employee_id: str, maximum: float) -> dict[str, object]:
    engine = RecommendationEngine(
        load_all_data(data_dir), absolute_max_skill_level=maximum,
    )
    return engine.recommend(
        employee_id, cutoff_date=date(2026, 10, 1),
    ).to_dict()
```

Результат содержит employee_id, current_level, target_level, skill_gaps, recommendations.
Каждая рекомендация: event_id, title, priority, evidence.
Evidence содержит дату, целевую роль/грейд, critical_gap, penalty, fallback,
развитие каждого навыка и коды причин fallback.
Порядок списка окончательный: сортировка только по priority на клиенте нарушит приоритет критических разрывов.
Название мероприятия отображается как в dataset; двуязычные объяснения генерируются отдельно.

Входные доменные модели допускают прямое создание в Python, поэтому внешние данные должны проходить через импортёр.
HTTP-контракт будет определён при добавлении REST API.
