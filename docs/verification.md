# Результаты проверки

База: main 49daa78. Рабочая ветка: codex/integrate-career-quest.
При начале работ незакоммиченных изменений не было. Main не изменялась.
Все 7 файлов data совпадают по Git blob hash с codex/backend-data ed1d355.

| Проверка | Результат |
|---|---|
| go test ./... | PASS: импорт, официальный dataset, progress, recommendation, service, store, HTTP, explanation |
| go vet ./... | PASS |
| go build ./... | PASS |
| gofmt для cmd и internal | Выполнен |
| python -m unittest backend.test_backend -v | PASS, 6 тестов; один проходит все 200 профилей |
| python -m compileall -q backend | PASS |
| python -m ruff check backend | PASS |
| python -m ruff format --check backend | PASS, 6 файлов |
| pnpm typecheck | PASS |
| pnpm test | PASS, 2 теста HTTP-клиента |
| pnpm build | PASS |
| CQ_TEST_URL=http://127.0.0.1:18080 node --test tests/api.integration.mjs | PASS: настоящий клиент → Go → completion → профиль и HR |
| Браузер, кнопка завершения | PASS: Cloud Certification Prep, прогресс 60,61% → 66,67%, добавлена история |
| Браузер, переключение KZ/HR | PASS: казахские объяснения, 200 сотрудников и реальные агрегаты |
| git diff --check | PASS |

Сквозные изменения выполнялись в памяти отдельного тестового экземпляра. Dataset на диске не менялся.

Не проверено:

- Go race detector: сначала CGO_ENABLED=0; при CGO_ENABLED=1 команда сообщает, что gcc отсутствует в PATH.
  Обычный тест конкурентного чтения/завершения проходит, но не заменяет race detector.
- Реальная внешняя LLM: отсутствуют URL и ключ. HTTP-контракт проверен локальным сервером; шаблоны, ошибка и timeout проверены тестами.
- Production-auth и сохранение после перезапуска не реализованы в исходном MVP: используются demo-заголовки и память.

## Изменённые пути относительно main

| Точный путь | Причина |
|---|---|
| `.env.example` | Явные настройки Go-сервера, среза, статического UI и необязательного LLM gateway. |
| `.gitignore` | Исключить локальные инструменты, кэши и артефакты проверок. |
| `README.md` | Единый запуск приложения, актуальные API и команды проверок. |
| `backend/ai_agent.py` | Корректное offline-объяснение fallback; убрать неиспользуемое чтение API key. |
| `backend/data_loader.py` | Строгий разбор фактической схемы events и role_profiles; проверка типов и ссылок. |
| `backend/models.py` | Типизированные Event и EventSkillGain с каноническими полями dataset. |
| `backend/recommendation_engine.py` | Offline-расчёт по реальным profiles/gains/history, явной дате и обязательным правилам. |
| `backend/test_backend.py` | Проверка официальных файлов, 200 профилей, caps, fallback и контрпримера жюри. |
| `cmd/career-quest/main.go` | Собрать единый Go-процесс API + web + объяснения, без Python runtime. |
| `data/README.kz.md` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/README.md` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/README.ru.md` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/activity_history.csv` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/employees.json` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/events.json` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `data/skills.json` | Перенести официальный синтетический dataset/описание из ed1d355 без изменения содержимого. |
| `docs/backend-development-guide.md` | Сохранить исходный guide и указать актуальный контракт интеграции. |
| `docs/development-plan.md` | Сохранить исходный план и исправить устаревший лимит LLM. |
| `docs/integration.md` | Решение архитектуры, факты о ветках, реальная схема, prerequisites, LLM-контракт и ограничения. |
| `docs/verification.md` | Результаты проверок и полный список путей с причиной изменения. |
| `go.mod` | Восстановить исходный Go-модуль 1.27.1; внешние Go-зависимости не добавлены. |
| `internal/config/config.go` | Восстановить конфигурацию Go из backend-ветки с фиксированной датой среза. |
| `internal/domain/contracts.go` | Сохранить camelCase HTTP DTO, добавить fallback/critical evidence, cutoff и HR counters. |
| `internal/domain/types.go` | Исходная схема dataset, максимум 5 и необязательные event prerequisites. |
| `internal/explanation/explanation.go` | ru/kk шаблоны, общий бюджет 1,4 секунды, защита score/evidence от провайдера. |
| `internal/explanation/explanation_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/explanation/http.go` | Необязательный HTTP text gateway с контекстом и таймаутом. |
| `internal/importer/importer.go` | Восстановить импорт Go, проверять канонические поля, дубли gains и ссылки prerequisite_events. |
| `internal/importer/importer_test.go` | Восстановить существующие Go-тесты из backend-ветки. |
| `internal/importer/official_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/progress/edge_cases_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/progress/progress.go` | Единый EffectiveGain: оба потолка, запрет понижения, начисления после review. |
| `internal/progress/progress_test.go` | Восстановить существующие Go-тесты из backend-ветки. |
| `internal/recommendation/edge_cases_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/recommendation/recommender.go` | Eligibility, caps, fallback, абсолютная critical-ступень и детерминированная сортировка. |
| `internal/recommendation/recommender_test.go` | Восстановить существующие Go-тесты из backend-ветки. |
| `internal/service/career.go` | HTTP use cases через Go-движки, снимок состояния, локализация, срез истории и реальные HR-агрегаты. |
| `internal/service/career_test.go` | Восстановить существующие Go-тесты из backend-ветки. |
| `internal/service/integration_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/store/memory.go` | Глубокие снимки данных под блокировкой и сохранение завершений. |
| `internal/store/memory_test.go` | Добавить регрессионные проверки реального формата и исправляемых граничных случаев. |
| `internal/transport/httpapi/handler.go` | Восстановить HTTP API, добавить каталог skills и locale ru/kk. |
| `internal/transport/httpapi/handler_test.go` | Восстановить существующие Go-тесты из backend-ветки. |
| `internal/transport/httpapi/web.go` | Раздавать собранный frontend тем же Go-сервером. |
| `pyproject.toml` | Воспроизводимые правила Ruff для Python. |
| `requirements-dev.txt` | Зафиксировать версию Ruff. |
| `web/.gitignore` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/index.html` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/package.json` | Сохранить React/Vite, добавить TypeScript и команды проверок. |
| `web/pnpm-lock.yaml` | Зафиксировать разрешённые версии зависимостей с TypeScript. |
| `web/pnpm-workspace.yaml` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/App.jsx` | Заменить runtime mockApi реальным Go API; обновлять профиль после completion и защищать загрузку от устаревших ответов. |
| `web/src/components/ActivityHistory.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/components/LanguageSwitch.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/components/MetricCard.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/components/ProgressBar.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/components/RecommendationCard.jsx` | Уникальные React keys для нескольких skill evidence. |
| `web/src/components/RoleSwitch.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/components/SkillProgress.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/i18n/translations.js` | ru/kk для настоящих статусов и ошибок; убрать ложные утверждения из демо. |
| `web/src/main.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/mocks/data.js` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/pages/CareerPath.jsx` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/pages/EmployeeDashboard.jsx` | Реальные инициалы сотрудника вместо имени из mock. |
| `web/src/pages/HRDashboard.jsx` | Реальные HR-данные, деление без нулевого знаменателя, общий объём регистраций. |
| `web/src/services/api.ts` | Типизированный HTTP-клиент и отображение серверных DTO без пересчёта рекомендаций. |
| `web/src/services/contracts.ts` | TypeScript-контракт фактических Go HTTP DTO и событий. |
| `web/src/services/mockApi.js` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/src/styles/index.css` | Восстановить этот файл из UI-ветки e424ccc; mocks сохранены как образцы и не импортируются приложением. |
| `web/tests/api.integration.mjs` | Проверить HTTP-клиент и полный сценарий на настоящем Go API. |
| `web/tests/api.test.mjs` | Проверить HTTP-клиент и полный сценарий на настоящем Go API. |
| `web/tsconfig.json` | Строгая проверка API-клиента. |
| `web/vite.config.js` | Проксировать /api и /health к Go в режиме разработки. |
