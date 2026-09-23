# Career Quest

Карьерные рекомендации для сотрудников и агрегированная статистика HR на официальном синтетическом dataset. Один Go-сервер загружает данные, рассчитывает навыки и рекомендации, сохраняет завершения в памяти и раздаёт React-интерфейс.

## Запуск

Нужны Go 1.27.1+, Node.js 24+ (для тестов TypeScript без отдельной компиляции) и pnpm.

```sh
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
go run ./cmd/career-quest
```

Откройте [localhost:8080](http://localhost:8080). Для запуска приложения Python не нужен.
Если pnpm отсутствует, его можно установить через `npm install -g pnpm@11.19.0`.
Рекомендуется использовать pnpm и существующий lockfile.

Переменные перечислены в [.env.example](.env.example). Приложение читает окружение процесса;
файл `.env` автоматически не загружается.
`SNAPSHOT_DATE=2026-10-01` — фиксированная дата по умолчанию, явно передаваемая в бизнес-логику.
Системное время не используется для расчётов. Максимум навыка — 5 согласно `data/skills.json`.

В режиме разработки запустите Go-сервер и `pnpm dev` из `web/`.
Vite направляет `/api` на Go по адресу `http://127.0.0.1:8080`.
В production-сборке UI и API обслуживает один Go-процесс.

## Архитектура

```text
data/*.json + activity_history.csv
  → internal/importer → internal/store
  → internal/progress + internal/recommendation
  → internal/service → internal/transport/httpapi
  → web/src/services/api.ts → React Employee / HR
                         ↘ internal/explanation: ru/kk template or optional text gateway
```

`backend/` сохранён как автономная Python-реализация для проверки данных и алгоритма.
Go не вызывает Python и не требует отдельно запускаемого сервиса.
Формат dataset описан в [data/README.ru.md](data/README.ru.md), решения интеграции — в [docs/integration.md](docs/integration.md).

## API

| Метод и путь | Назначение |
|---|---|
| GET /health | Состояние и размеры dataset |
| GET /api/employees | Список сотрудников для HR |
| GET /api/v1/employees/{id}/profile | Профиль, эффективные навыки, gaps, evidence и рекомендации |
| POST /api/v1/employees/{id}/complete | Завершение: JSON `{"eventId":"EV_005"}`, ответ с пересчитанным профилем |
| GET /api/events | Каталог с исходными полями dataset |
| GET /api/skills | Названия навыков для интерфейса |
| GET /api/v1/hr/analytics | HR-агрегаты |
| POST /api/import | Импорт четырёх файлов multipart или ZIP |

Demo-доступ: `X-Demo-Role: employee` и `X-Employee-ID: E0001`, либо `X-Demo-Role: hr`.
Это демонстрационное разграничение, не аутентификация для публичного production-развёртывания.
UI по умолчанию показывает `E0001`; HR получает агрегаты всех 200 сотрудников.
Язык объяснений: `?locale=ru` или `?locale=kk`.
Изменения хранятся в памяти до перезапуска. Исходный dataset не перезаписывается.

## Правила

- Поля мероприятий: `target_roles`, `target_grades`, `develops_skills`.
- `prerequisites` — уровни навыков. Необязательное расширение `prerequisite_events` — ID мероприятий, требующих completed до даты среза включительно.
- Начисляются только завершения после `last_review_date` и не позже даты среза.
- Прирост ограничен потолком мероприятия и 5; достигнутый уровень никогда не понижается.
- Критический gap — отдельная ступень сортировки выше любого некритического кандидата, независимо от штрафов.
- При нулевом gap работают поддержание hard-навыков, soft-skills и менторство; все фильтры доступности сохраняются.
- Если допустимых событий нет вообще, возвращается пустой список с соответствующим текстом, а не вымышленное мероприятие.
- LLM может менять только текст; общий бюджет ожидания для всех объяснений ответа — 1,4 секунды, затем готовый шаблон.

## Проверки

```sh
go test ./...
go vet ./...
go build ./...
python -m unittest backend.test_backend -v
python -m pip install -r requirements-dev.txt
python -m ruff check backend
python -m ruff format --check backend
cd web
pnpm typecheck
pnpm test
pnpm build
```

Сквозной тест на отдельном свежем экземпляре сервера, PowerShell:

```powershell
# Терминал 1, корень проекта
$env:APP_PORT = '18080'
go run ./cmd/career-quest
# Терминал 2, каталог web
$env:CQ_TEST_URL = 'http://127.0.0.1:18080'
node --test tests/api.integration.mjs
```

Тест выполняет реальное завершение в памяти этого экземпляра. Для повторного запуска перезапустите сервер.
