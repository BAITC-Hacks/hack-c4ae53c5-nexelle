# Career Quest

Career Quest подбирает сотруднику объяснимые шаги развития по карьерной цели, навыкам и истории активностей. HR получает агрегаты по пробелам навыков и участию в мероприятиях.

## Запуск

Нужен Docker Desktop и каталог `data/` с `skills.json`, `employees.json`, `events.json` и `activity_history.csv`.

```powershell
docker compose up --build
```

- UI: <http://localhost:3000>
- API: <http://localhost:8080>
- дата среза: `2026-10-01`

UI обращается к API с demo-заголовками `X-Demo-Role` и `X-Employee-ID`. Роль сотрудника использует профиль `E0001`; переключатель HR запрашивает агрегированный срез.

## Проверка без Docker

```powershell
go test ./...
go build ./...
go vet ./...

cd web
corepack pnpm install --frozen-lockfile
corepack pnpm build
```

## Ключевые API

- `GET /api/employees/{id}` — профиль, эффективные навыки, разрывы и рекомендации;
- `POST /api/employees/{id}/complete` — завершение мероприятия и пересчёт;
- `GET /api/hr/analytics` — HR-аналитика;
- `POST /api/import` — горячая загрузка и валидация датасета.
