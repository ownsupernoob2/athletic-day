# Athletics Day

Replaces the "Excel by email, then build PDFs by hand" process.

| Who | What they do |
|---|---|
| **Staff** | Sign in, see their group's students, tap events (auto-saves, minimum 3 per student), add a missing student, download their group CSV |
| **Organiser** | See which groups have finished, manage students (search, add, edit, remove, CSV import), staff accounts, events and rules, build/randomise/edit/publish the schedule, download everything as a zip |
| **Everyone** (no login) | `/schedule`: search the published schedule, download it as CSV, print it as a booklet |

```
server/   Go API (net/http + SQLite, no CGO)
web/      Next.js 16 frontend (Tailwind), proxies /api/* to the Go server
old/      the original Flask MVP (reference only)
```

## Run it

Two terminals.

```bash
# 1. API on :8080 (SEED_DEMO=1 fills an empty database with fake students and staff)
cd server
SEED_DEMO=1 go run .

# 2. Website on :3000
cd web
npm install
npm run dev
```

Open http://localhost:3000.

First organiser login (created only when the database is empty):
`organiser@school.test` / `changeme123`. Set your own with `ADMIN_EMAIL` and `ADMIN_PASSWORD`.
Demo staff logins: `staff.ssh@school.test`, `staff.tdr@school.test`, ... password `staffpass123`.

### Server settings (environment variables)

| Variable | Default | |
|---|---|---|
| `PORT` | `8080` | |
| `DB_PATH` | `athletic.db` | the SQLite file. Back this up |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | see above | first organiser, only used on an empty database |
| `SEED_DEMO` | off | `1` loads demo data (only on an empty database) |
| `COOKIE_SECURE` | off | set `1` when served over HTTPS |
| `KAMAR_USERNAME` / `KAMAR_PASSWORD` | unset | the login KAMAR must send. Unset means KAMAR is refused. Put them in `server/.env` (see `.env.example`), never in git |

The website finds the API through `API_URL` (default `http://localhost:8080`).

## Rules in the app

- Group codes are 3 letters. The **first letter is the house**: S Stuart, T Tudor, W Windsor, H Hanover.
- Age groups (Under 14, 15, 16, 19, Over 19) are worked out on the **athletics day date** set on the Events page.
- Heats hold up to the configured size (default 6) and are balanced, so 7 runners become 4 + 3, never 6 + 1.
- Export format matches the old app's `formatted.csv`: five header rows, then `Last, First, Group, Age group` and one column letter per event.
  The zip has one folder per house (`Stuart/SSH.csv`, ...).
- CSV import columns: `First Name, Last Name, DOB, Group ID` (like `old/data.csv`).

## KAMAR (student list)

`POST /api/kamar` is a KAMAR Directory Services listener (`server/kamar.go`). The school enters the URL, username and password in KAMAR and it pushes the student list: preferred first name, last name, date of birth and tutor group only. Hand the school [docs/kamar-setup.md](docs/kamar-setup.md).
In production point KAMAR at the Go server (over HTTPS), not through the Next.js proxy, which can limit large request bodies.

## Auth0 / Microsoft sign-in (planned)

Sign-in is currently email + password with accounts made by the organiser (`server/auth.go`).
To move to Auth0: verify the Auth0 token in `currentUser()` and map the Microsoft account's email to a row in `users`.
Everything else (roles, group scoping) stays as is.
