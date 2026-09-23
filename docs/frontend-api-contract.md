# Frontend API contract

## Purpose and transport boundary

The React UI imports only the functions exported from `web/src/services/api.js`. The active implementation is selected by `VITE_API_MODE`:

- `mock` (default) uses `mockApi.js` and persists demo completion state in browser `localStorage`.
- `real` uses `apiClient.js` and the proposed HTTP paths below.

The UI does not call `fetch` directly. In real mode, `VITE_API_BASE_URL` is required. The backend integration team may change the endpoint paths in `api.js` without changing pages or components.

## Service functions and proposed paths

| Frontend function | Method and path | Expected result |
| --- | --- | --- |
| `getEmployeeProfile(employeeId)` | `GET /employees/{employeeId}/profile` | Employee profile DTO |
| `getEmployeeHistory(employeeId)` | `GET /employees/{employeeId}/history` | Array of activity history entries |
| `getRecommendations(employeeId)` | `GET /employees/{employeeId}/recommendations` | Array of recommendation DTOs |
| `completeActivity(employeeId, eventId)` | `POST /employees/{employeeId}/activities/{eventId}/complete` | Successful response; frontend then reloads profile, history, and recommendations |
| `getHRDashboard()` | `GET /hr/dashboard` | HR dashboard DTO |

`completeActivity` may return `204 No Content` or JSON. The frontend intentionally re-reads the affected data after a successful request, so it does not require a mutation response body.

## Employee profile DTO

```json
{
  "employee_id": "emp-003",
  "name": "Айым Сейтова",
  "role": "Product Analyst",
  "grade": "G2 · Middle",
  "career_goal": {
    "target_role": "Senior Product Analyst",
    "target_grade": "G3 · Senior",
    "target_date": "Q4 2026"
  },
  "skills": [
    {
      "name": "SQL",
      "current_level": 2,
      "required_level": 4,
      "gap": 2
    }
  ],
  "progress": {
    "completed_activities": 3,
    "total_activities": 7,
    "percentage": 43
  },
  "history": [
    {
      "activity_id": "hist-1",
      "title": "Product Metrics Foundations",
      "format": "Course",
      "date": "18.08.2026",
      "status": "completed"
    }
  ]
}
```

`history` is supported both inside the profile and through `getEmployeeHistory(employeeId)`. The UI uses the dedicated history response when it is available.

## Recommendation DTO

```json
{
  "event_id": "event-sql-lab",
  "title": "Advanced SQL: Window Functions Lab",
  "event_type": "Workshop",
  "availability": {
    "date": "26.09.2026 · 15:00",
    "status": "available"
  },
  "skill_gaps": ["SQL"],
  "expected_gain": [{ "skill": "SQL", "level_delta": 1 }],
  "prerequisites": ["Базовый SQL"],
  "eligibility": { "status": "eligible" },
  "history_signals": ["Продолжает тему SQL после базового курса"],
  "score_breakdown": {
    "grade": 24,
    "skill_gaps": 30,
    "career_goal": 22,
    "activity_history": 12,
    "availability": 12
  },
  "explanation": {
    "why": "Практическая лаборатория закрывает самый большой skill gap.",
    "evidence": [
      { "factor": "skill_gaps", "detail": "SQL — разрыв 2 уровня из требуемых 4." }
    ]
  }
}
```

`event_type` is the contract field. `RecommendationCard` temporarily reads legacy `format` as a backwards-compatible fallback, but the mock DTO now uses `event_type`. `prerequisites` is top-level; the card also accepts the former nested mock shape as a temporary fallback.

## HR dashboard DTO

```json
{
  "total_employees": 128,
  "employees_in_development": 83,
  "employees_without_recommendation": 14,
  "skill_gaps": [
    { "skill": "SQL", "employees": 36 }
  ],
  "activity_participation": {
    "completed": 67,
    "no_show": 8,
    "dropped": 11,
    "declined": 19
  }
}
```

## Current mock differences

- Mock persistence (`completed` on an internal recommendation record and browser `localStorage`) is demo-only and is not part of the backend DTO.
- `date: "today"` is a mock sentinel used by the UI to localize an activity completed in the current session. A real API should return a display-ready date or an agreed machine-readable date format.
- No authentication, authorization, pagination, filtering, or error-envelope contract is defined yet. These need alignment before production integration.
