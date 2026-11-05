# Athletics Day

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
