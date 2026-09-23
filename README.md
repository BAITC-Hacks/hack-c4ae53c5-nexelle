# Career Quest

Career Quest показывает сотруднику карьерную цель, skill gaps и следующий шаг развития с объяснением. HR получает агрегированную картину развития команды.

## MVP

- Профиль сотрудника с assessed skills, effective skills, целью и readiness.
- До трёх рекомендаций с Evidence DTO и разложением score.
- Completion обновляет навыки и пересчитывает траекторию в памяти текущей сессии.
- HR-панель с пробелами навыков и показателями участия.
- UI с demo-ролями, RU/KK и режимами mock/HTTP.

## Запуск backend

Требуется Go 1.27+ и каталог с `skills.json`, `employees.json`, `events.json`, `activity_history.csv`.

```powershell
$env:DATA_DIR = '.\data'
$env:APP_PORT = '8080'
go run ./cmd/career-quest
```

Проверка состояния: `Invoke-WebRequest http://localhost:8080/health`.

Основные endpoints:

- `GET /api/v1/employees/{id}/profile` - профиль, progress и рекомендации;
- `POST /api/v1/employees/{id}/complete` - completion с телом `{"eventId":"EV_005"}`;
- `GET /api/v1/hr/analytics` - HR-агрегаты;
- `GET /api/v1/dataset/stats` - статистика датасета.

Для employee передавайте `X-Demo-Role: employee` и свой `X-Employee-ID`. Для HR - `X-Demo-Role: hr`. API использует camelCase DTO; CORS разрешён для `localhost:3000`, `5173` и `8080`.

## Запуск UI

```sh
go run ./web/server.go
```

Откройте <http://localhost:4173>. Mock API включён по умолчанию и хранит completion в `localStorage`; кнопка «Сбросить демо» восстанавливает начальное состояние. Режим HTTP включается параметром `?mock=false` после подключения актуальных путей API в `web/src/api.js`.

Профиль доступен HR либо самому сотруднику. Для demo-запроса сотрудника передайте оба заголовка:

```powershell
Invoke-WebRequest http://localhost:8080/api/v1/employees/E0001/profile -Headers @{
  'X-Demo-Role' = 'employee'
  'X-Employee-ID' = 'E0001'
}
```

## Проверка

```powershell
go fmt ./...
go test ./...
go build ./...
go vet ./...
```

Конфигурация приведена в `.env.example`; секреты не добавляются в репозиторий.
