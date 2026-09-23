> Исторический план разработки. Актуальные запуск, HTTP-контракт и интеграция описаны в README.md и docs/integration.md.

# Career Quest - пошаговая инструкция для backend-разработчика

## Результат твоей зоны

К моменту интеграции backend должен уметь:

1. Загружать исходные `skills.json`, `employees.json`, `events.json`, `activity_history.csv` и дополнительные проверочные данные в той же схеме.
2. Отдавать сотрудника, его историю, карьерную траекторию и данные для UI.
3. Принимать действие «мероприятие завершено» и сохранять новую запись истории.
4. Пересчитывать эффективные навыки и получать свежую рекомендацию из модуля участника 2.
5. Отдавать HR-агрегаты и не раскрывать данные других сотрудников обычному сотруднику.

Backend не должен сам генерировать текст с помощью LLM и не должен дублировать scoring из `internal/recommendation/`: он вызывает согласованный интерфейс recommendation-модуля.

## Перед началом кодирования - 30 минут

### 1. Зафиксируй договорённости с командой

До создания обработчиков согласуйте один файл API-контракта. В нём зафиксируйте:

- формат идентификаторов: `employee_id`, `event_id`, `skill_id`, `record_id`;
- как передаётся роль вызывающего: `employee` или `hr`;
- DTO ответа профиля, рекомендации, completion и HR-среза;
- текстовые коды ошибок, которые UI должен обрабатывать;
- кто владеет каждым пакетом, чтобы не редактировать одни файлы параллельно.

Минимальный контракт с модулем рекомендаций:

```go
type RecommendationService interface {
    Recommend(ctx context.Context, input RecommendationInput) (RecommendationResult, error)
    BuildTrajectory(ctx context.Context, input TrajectoryInput) (Trajectory, error)
}
```

`RecommendationInput` не должен содержать HTTP-объекты, секреты или глобальное состояние. В него входят уже загруженные профиль, каталог навыков, профили ролей, мероприятия, история и дата среза.

### 2. Выбери только утверждённые командой технологии

Обязательные условия: Go 1.27+, переменные окружения для конфигурации и один воспроизводимый запуск. Конкретные HTTP-фреймворк, БД, клиентский формат и LLM-провайдер выбираются всей командой. До этого можно начать с интерфейсов Go и тестов без внешних зависимостей.

## Шаг 1. Создай каркас проекта

Создай пакеты так, чтобы зоны участников не пересекались:

```text
cmd/career-quest/             # точка входа, сборка зависимостей
internal/config/              # чтение конфигурации
internal/domain/              # структуры предметной области, без HTTP/БД
internal/importer/            # JSON/CSV import и validation report
internal/store/               # интерфейсы хранилища и реализация для демо
internal/service/             # use cases: profile, completion, hr view
internal/transport/http/      # маршруты, middleware, request/response DTO
internal/recommendation/      # владелец: участник 2
internal/progress/            # владелец: участник 2
internal/explanation/         # владелец: участник 2
```

Правила:

- `cmd/` только собирает зависимости и запускает сервер; бизнес-логики в `main.go` нет.
- `domain` не импортирует transport, storage или конкретную БД.
- Любая операция чтения/записи принимает `context.Context` первым параметром.
- Ошибки возвращаются наверх; `panic` разрешён только для невозможной ошибки инициализации приложения, но не для данных пользователя.

Сразу добавь `.env.example` без ключей. Минимальные переменные:

```dotenv
APP_PORT=8080
DATA_DIR=./data
SNAPSHOT_DATE=2026-10-01
DEMO_ROLE=hr
```

Значение даты среза должно быть настраиваемым, но по умолчанию соответствовать датасету. Не используй `time.Now()` для бизнес-расчётов.

## Шаг 2. Опиши доменные структуры

Создай типы в `internal/domain`. Называй поля в Go идиоматично, а теги JSON оставляй в формате исходного датасета.

Обязательные сущности:

- `Skill`: `SkillID`, `Name`, `Type`, `Category`, `Description`.
- `RoleProfile`: `Role`, `Grade`, `RequiredSkills map[string]int`, `CriticalSkills []string`.
- `CareerGoal`: `TargetRole`, `TargetGrade`.
- `Employee`: профиль из файла, включая `Skills map[string]int`, `LastReviewDate`, nullable `ManagerID` и nullable `CareerGoal`.
- `Event`: аудитория, формат, `Mandatory`, развиваемые навыки, prerequisites и будущие сессии.
- `EventSkillGain`: `SkillID`, `Gain`, `MaxLevel`.
- `ActivityRecord`: `RecordID`, `EmployeeID`, `EventID`, дата, deadline, статус, completion, score, feedback, initiator.
- `Dataset`: все загруженные сущности и индексы для быстрого доступа.
- `Actor`: роль сессии и `EmployeeID`, если запрос выполняется сотрудником.

Для статусов и ролей используй строковые константы, а не свободный текст во всех обработчиках:

```go
const (
    ActivityCompleted  = "completed"
    ActivityInProgress = "in_progress"
    ActivityDropped    = "dropped"
    ActivityNoShow     = "no_show"
    ActivityDeclined   = "declined"
    ActivityOverdue    = "overdue"

    RoleEmployee = "employee"
    RoleHR       = "hr"
)
```

Не храни вычисленные effective skills в исходном `Employee.Skills`: это оценка на `last_review_date`. Вычисленная версия должна быть отдельной структурой, возвращаемой progress-модулем.

## Шаг 3. Реализуй загрузку и валидацию датасета

### 3.1 Интерфейс импортера

Сделай единый use case, который принимает источник данных и возвращает либо готовый `Dataset`, либо подробный отчёт ошибок:

```go
type DatasetImporter interface {
    Load(ctx context.Context, source DatasetSource) (domain.Dataset, ValidationReport, error)
}

type ValidationReport struct {
    Errors   []ValidationIssue `json:"errors"`
    Warnings []ValidationIssue `json:"warnings"`
}

type ValidationIssue struct {
    File    string `json:"file"`
    Row     int    `json:"row,omitempty"`
    Field   string `json:"field,omitempty"`
    Message string `json:"message"`
}
```

Ошибки формата должны быть понятны HR и жюри: например, `employees.json: employee E9999 refers to missing manager E0000`.

### 3.2 Порядок загрузки

1. Прочитай `skills.json`: каталог навыков, шкалу и 32 профиля ролей (`8 x 4`).
2. Прочитай `events.json` и построй индекс `event_id -> Event`.
3. Прочитай `employees.json` и построй индекс `employee_id -> Employee`.
4. Прочитай `activity_history.csv` потоково через `encoding/csv`; не предполагая, что файл останется размером стартового кита.
5. Построй индексы: записи по сотруднику, записи по мероприятию, профили по `role + grade`.

### 3.3 Что обязательно валидировать

- Все ID уникальны внутри своей сущности.
- Каждый `skill_id` в профилях ролей, навыках сотрудников, gains и prerequisites существует в каталоге навыков.
- Каждая пара `(role, grade)` сотрудника и карьерной цели существует среди `role_profiles`.
- `manager_id` либо пустой, либо указывает на сотрудника с грейдом `Lead` того же отдела.
- Уровень навыка сотрудника и требования лежат в диапазоне `0..5`.
- `gain > 0`, `max_level` в диапазоне `1..5`, а `gain` не повышает уровень выше `max_level` при применении.
- `event_id` и `employee_id` в истории существуют.
- Статус истории, `completion_pct`, score, feedback и `assigned_by` соответствуют правилам README.
- `completed` имеет `completion_pct == 100`; `no_show` и `declined` имеют `0`; для обязательных записей корректно заполнен `due_date`.
- Для не-self-paced мероприятия есть валидные даты сессий, а у self-paced сессии не требуются.

Не отклоняй весь импорт из-за одной ошибочной строки дополнительных данных. Собери все ошибки, не публикуй частичный датасет и верни `422 Unprocessable Entity` с `ValidationReport`.

### 3.4 Импорт тестовых данных жюри

Сделай отдельный сценарий загрузки, который добавляет или заменяет только указанные сущности. Он не должен перезаписывать стартовые файлы на диске. До применения:

1. Провалидируй новые профили и записи вместе с базовым `Dataset`.
2. Убедись, что нет конфликта ID, если режим - `append`.
3. В режиме `replace_employee` разреши заменить только профиль указанного сотрудника и его историю.
4. Примени изменения атомарно: при ошибке активный dataset не меняется.
5. Верни количество добавленных/заменённых сущностей и полный validation report.

## Шаг 4. Сделай хранилище и use cases

Сначала определи интерфейс, потом выбери реализацию. Для демонстрации достаточна потокобезопасная реализация в памяти, которая держит импортированный набор и новые ActivityRecord. Если команда выберет БД, она должна реализовать тот же контракт.

```go
type Store interface {
    GetEmployee(ctx context.Context, id string) (domain.Employee, error)
    ListEmployeeActivities(ctx context.Context, employeeID string) ([]domain.ActivityRecord, error)
    GetEvent(ctx context.Context, id string) (domain.Event, error)
    GetDataset(ctx context.Context) (domain.Dataset, error)
    ReplaceDataset(ctx context.Context, dataset domain.Dataset) error
    AddActivity(ctx context.Context, record domain.ActivityRecord) error
}
```

Проверь `context.Context` до долгих операций и защищай in-memory состояние `sync.RWMutex`. Не возвращай наружу внутренние `map` или `slice`: копируй их, чтобы обработчик не мог изменить хранилище без явного метода.

Собери service-слой:

- `ProfileService.GetProfile`: читает данные сотрудника, вызывает progress и recommendation, собирает DTO.
- `CompletionService.CompleteEvent`: проверяет доступ, событие и дубликат, добавляет completed record, затем заново строит профиль.
- `HRService.GetOverview`: агрегирует пробелы, отсутствие рекомендаций и статистику участия.
- `ImportService.Import`: вызывает importer и атомарно заменяет/дополняет dataset.

## Шаг 5. Реализуй completion без обхода правил

Endpoint completion не принимает от UI уровни навыков и итоговый score. UI передаёт только `event_id`; сервер сам берёт gain и max_level из каталога.

Порядок операции:

1. Получить `Actor` из middleware.
2. Запретить сотруднику завершать активность за другого сотрудника.
3. Проверить существование employee и event.
4. Отклонить `mandatory` как действие карьерной рекомендации, если команда не согласовала отдельный сценарий compliance.
5. Проверить, что событие не было `completed` ранее; исключение - `EV_036`.
6. Проверить prerequisites по effective skills на момент completion.
7. Создать серверный `ActivityRecord` со статусом `completed`, `completion_pct: 100`, текущей датой в рамках demo-сценария и уникальным `record_id`.
8. Сохранить запись в одной транзакции/критической секции.
9. Вызвать progress/recommendation с обновлённой историей.
10. Вернуть новые effective skills, изменившиеся gaps, trajectory и 1-3 новых рекомендации.

Важно: completion может менять только effective skills, так как исходная оценка привязана к `last_review_date`. В response явно раздели `assessed_skills` и `effective_skills`, чтобы не создавать ложное впечатление о новой HR-аттестации.

## Шаг 6. Спроектируй HTTP API

Пути и поля можно скорректировать после общего контракта, но следующего минимума достаточно для MVP.

| Метод и путь | Доступ | Назначение |
|---|---|---|
| `GET /health` | все | Статус приложения и активного набора данных. |
| `GET /api/v1/employees` | HR | Список сотрудников для HR-экрана; без полной истории. |
| `GET /api/v1/employees/{id}/profile` | HR или этот employee | Профиль, trajectory, skills, history summary, recommendations и evidence. |
| `POST /api/v1/employees/{id}/activities/{eventID}/complete` | HR или этот employee | Отметить добровольное мероприятие завершённым и вернуть пересчёт. |
| `GET /api/v1/hr/overview` | HR | Агрегаты пробелов, рекомендаций и участия. |
| `POST /api/v1/imports` | HR | Импорт проверочных файлов и validation report. |

Ответ ошибки единообразен:

```json
{
  "error": {
    "code": "event_already_completed",
    "message": "Мероприятие уже завершено сотрудником",
    "details": []
  }
}
```

Минимальная карта кодов:

- `400 invalid_request` - поле запроса не прошло синтаксическую проверку;
- `401 unauthenticated` - нет demo-сессии/токена;
- `403 forbidden` - роль не имеет доступа либо employee запросил коллегу;
- `404 employee_not_found`, `event_not_found`;
- `409 event_already_completed`, `dataset_not_loaded`;
- `422 import_validation_failed`, `event_prerequisites_not_met`;
- `500 internal_error` - без внутренних деталей в ответе.

DTO профиля обязан явно различать:

- исходные `assessed_skills`;
- `effective_skills` после завершений после `last_review_date`;
- `target` и `skill_gaps`;
- `recommendations` с `event`, `eligibility`, `score_breakdown`, `evidence` и `explanation`.

Не передавай в employee-response список других сотрудников, manager-only заметки или сырые HR-агрегаты.

## Шаг 7. Добавь разграничение доступа

Для MVP допустима demo-аутентификация через middleware, если в README прозрачно указано, что это не production SSO. Middleware создаёт `Actor` и кладёт его в `context.Context`.

Проверки должны быть на service-слое, а не только спрятаны в UI:

- `employee` читает и меняет только запись с собственным `employee_id`;
- `hr` получает HR-view, импорт и доступ к профилям;
- отсутствие или неизвестная роль отклоняется;
- все действия с импортом и completion логируются с actor, временем и target id без секретов.

Не храните персональные данные за пределами проекта, не отправляйте исходные профили во внешний LLM-провайдер и не добавляйте публичный рейтинг сотрудников.

## Шаг 8. Собери HR-агрегаты

HR-экрану не нужны необъяснимые «оценки человека». Отдавай агрегируемые, проверяемые показатели:

- `top_skill_gaps`: навыки с числом сотрудников, имеющих разрыв до цели;
- `employees_without_recommendation`: ID/имя, причина отсутствия кандидата (нет доступных событий, prerequisites, Lead без цели и т. п.);
- `participation_by_event`: регистрации, completed, no_show, dropped, declined, completion rate;
- фильтры по роли, грейду и отделу;
- дата среза и время последнего импорта.

Агрегат считается на основании тех же effective skills и фильтров, что и профиль. Нельзя считать слабые навыки только по `Employee.Skills`, иначе HR и сотрудник увидят разные результаты.

## Шаг 9. Напиши тесты до интеграции UI

### Импортер

- Загружает исходный датасет: 200 сотрудников, 40 событий, 60 навыков, 2 743 записи.
- Отклоняет неизвестный skill/event/employee ID с указанием файла и поля.
- Отклоняет duplicate IDs и неправильные статусы/проценты.
- При ошибке дополнительного импорта не меняет активный dataset.

### Access control

- Employee может открыть свой профиль и не может открыть профиль коллеги.
- Employee не получает `/hr/overview` и `/imports`.
- HR может загрузить данные и открыть профиль сотрудника.

### Completion

- После completion появляется один `completed` record с `completion_pct=100`.
- Уровень навыка не превышает `max_level`.
- Уже завершённое мероприятие отклоняется, кроме `EV_036`.
- Мероприятие с невыполненным prerequisite отклоняется.
- В ответе возвращается пересчитанная trajectory и рекомендации.

### HTTP

- Корректные статусы для `400`, `403`, `404`, `409`, `422`.
- DTO не содержит поля HR, когда его запрашивает employee.
- Повторный запрос completion не создаёт дубликат при конкурентных запросах.

Запускай после каждого изменения:

```powershell
gofmt -w .
go test ./...
go build ./...
```

Перед финальной защитой дополнительно прогони `go vet ./...`, если в выбранном стеке он применим.

## Шаг 10. Проведи интеграцию с командой

### Перед подключением UI

1. Подними сервер на локальном порту.
2. Отдай участнику 3 примеры успешных и ошибочных JSON-ответов.
3. Убедись, что один запрос профиля занимает меньше 2 секунд на стартовом датасете.
4. Дай участнику 2 один fixture с сотрудником, событиями и историей для regression-тестов.

### Сквозной demo-сценарий

1. HR импортирует базовый набор и, при необходимости, дополнительный проверочный профиль.
2. Employee открывает только собственный профиль.
3. Backend вызывает движок, UI показывает top-3 и facts из evidence.
4. Employee завершает доступное добровольное мероприятие.
5. Backend возвращает изменённые effective skills, gaps и рекомендации.
6. HR открывает overview и видит пересчитанные агрегаты.

Если LLM недоступна, сценарий должен пройти с шаблонным explanation. Это не повод блокировать backend или демонстрацию.

## Контрольный список перед передачей

- [ ] Датасет запускается из чистой рабочей директории по README.
- [ ] `SNAPSHOT_DATE=2026-10-01` не захардкожен внутри use cases.
- [ ] Никаких ключей и персональных данных нет в репозитории, логах или ошибках.
- [ ] Импорт дополнительных данных атомарен и выдаёт понятный report.
- [ ] Employee API не раскрывает коллег и HR-агрегаты.
- [ ] Completion серверный, идемпотентно защищён и не принимает skill level от клиента.
- [ ] Recommendation-модуль подключён только через интерфейс и возвращает evidence.
- [ ] `gofmt`, `go test ./...` и `go build ./...` проходят.
- [ ] README содержит команды запуска, пример загрузки и demo-сценарий.
