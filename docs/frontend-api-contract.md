# Frontend API contract

## Modes and service boundary

The React UI calls only the functions exported by `web/src/services/api.js`.
Components do not call `fetch` directly.

- `VITE_API_MODE=mock` is the default. It uses `mockApi.js` and retains the
  standalone demo's browser `localStorage` persistence.
- `VITE_API_MODE=real` uses the Go backend through `apiClient.js` and adapts
  its DTOs into the UI DTOs below.

Real mode requires `VITE_API_BASE_URL=http://localhost:8080`. The current
hackathon identity is `E0001`; it can be overridden with
`VITE_DEMO_EMPLOYEE_ID` until an authenticated identity provider replaces it.

## Current mock contract

The mock API remains the UI contract for pages and components:

| Function | Mock result |
| --- | --- |
| `getEmployeeProfile(employeeId)` | Employee DTO with `employee_id`, `name`, `career_goal`, `skills`, `progress`, and `history` |
| `getEmployeeHistory(employeeId)` | Activity array with `activity_id`, `title`, `format`, `date`, and `status` |
| `getRecommendations(employeeId)` | Recommendation array with `event_id`, `event_type`, score breakdown, eligibility, availability, evidence, and expected gains |
| `completeActivity(employeeId, eventId)` | Marks the activity complete and updates in-memory/localStorage mock state |
| `getHRDashboard()` | HR DTO with totals, skill gaps, and aggregated participation |

Mock mode does not make HTTP requests. Completion state is stored under
`career-quest.mock-api.v1` and therefore remains after a browser reload.

## Actual Go backend contract

The Go backend uses `http://localhost:8080` and demo authorization headers:

- Employee requests: `X-Demo-Role: employee` and `X-Employee-ID: E0001`
- HR requests: `X-Demo-Role: hr`

The service layer sends those headers; pages and components do not manage them.

| Frontend service function | Go endpoint used in real mode | Actual response |
| --- | --- | --- |
| `getEmployeeProfile(employeeId)` | `GET /api/v1/employees/{id}/profile` | `ProfileResponse` |
| `getEmployeeHistory(employeeId)` | `GET /employees/{id}/history` | Raw `ActivityRecord[]` |
| `getRecommendations(employeeId)` | `GET /api/v1/employees/{id}/profile` | Extracts `ProfileResponse.recommendations` |
| `completeActivity(employeeId, eventId)` | `POST /employees/{id}/complete` with `{"eventId":"..."}` | `CompletionResponse` |
| `getHRDashboard()` | `GET /hr/dashboard` | `HRAnalytics` |

The backend enables CORS for the Vite development origin
`http://localhost:5173` and permits these demo headers.

### Go profile and history DTOs

`ProfileResponse` has these relevant fields:

```json
{
  "employee": {
    "employeeId": "E0001",
    "fullName": "...",
    "role": "...",
    "grade": "...",
    "careerGoal": { "targetRole": "...", "targetGrade": "..." }
  },
  "activities": [{
    "recordId": "...",
    "eventId": "...",
    "eventTitle": "...",
    "eventFormat": "online",
    "date": "2026-10-01",
    "status": "completed"
  }],
  "progress": {
    "target": { "role": "...", "grade": "...", "source": "careerGoal" },
    "skillGaps": [{
      "skillId": "...",
      "skillName": "...",
      "currentLevel": 2,
      "requiredLevel": 4,
      "gap": 2,
      "isCritical": true
    }],
    "readinessPercent": 50
  },
  "recommendations": []
}
```

The dedicated history endpoint returns raw activity records with snake-case
fields such as `record_id`, `employee_id`, `event_id`, `date`, `status`, and
`completion_pct`. It does not include event title or format.

### Go recommendation and HR DTOs

Recommendations in the profile response use `eventId`, `title`, `format`,
`nextSessionDate`, `score`, `evidence`, `explanation`, and localized
`explanations`. The backend has already applied eligibility, prerequisites,
availability, and recommendation scoring; the frontend does not calculate or
rank recommendations.

`HRAnalytics` uses `employeeCount`, `topSkillGaps`,
`employeesWithoutRecommendation`, and `participationByEvent`. Each
participation item includes `completed`, `noShow`, `dropped`, and `declined`.

## Frontend adapter behavior

`api.js` maps real responses into the existing page/component DTOs:

- Employee `fullName`, camel-case career goal, `skillGaps`, and
  `readinessPercent` become the existing employee, skills, and progress
  fields. `completed_activities` and `total_activities` are derived from
  profile activities.
- Raw history records become `activity_id`, `title`, `format`, `date`, and
  `status`. The concurrently loaded profile supplies event title/format
  metadata. Backend formats map to existing UI labels as `self_paced` →
  `Course`, `online` → `Workshop`, and other formats → `Event`.
- `getRecommendations()` extracts the server-provided profile
  recommendations. It maps server scores and evidence without creating a
  frontend recommendation algorithm. The Go DTO has no displayable
  prerequisite list, so the adapter exposes an empty `prerequisites` array.
- HR skill-gap and per-event participation arrays are aggregated into the
  current dashboard DTO.
- Completion only posts the backend's required body. `App.jsx` retains its
  existing post-completion refetch of profile, history, and recommendations.

`apiClient.js` continues to throw `ApiClientError`; it now also reads the Go
error envelope at `error.message`, so existing UI load-error handling remains
unchanged.
