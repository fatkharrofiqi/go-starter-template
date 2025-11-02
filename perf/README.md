Performance tests (k6)

Overview:
- Scripts are separated from Go unit tests under `perf/`.
- Use `k6` to run scenarios manually; pass parameters via env vars.

Structure:
- `perf/load/`: normal traffic scenarios
- `perf/stress/`: beyond-capacity scenarios
- `perf/smoke/`: quick checks (add as needed)
- `perf/helpers/`: shared utilities (optional)

Naming convention:
- `resource.action.scenario.k6.js`
- Examples:
  - `users.list.load.k6.js`
  - `users.get.smoke.k6.js`
  - `users.get_with_csrf.stress.k6.js`

Environment variables (via `-e` or system env):
- `BASE_URL`: API base URL (default examples use `http://localhost:8080`)
- `AUTH_TOKEN`: bearer token if required by the script
- `CSRF_TOKEN`: CSRF token if the scenario needs it

Run examples:
- List users (load):
  - `k6 run -e BASE_URL=http://localhost:8080 perf/load/users.list.k6.js`
- Get user (load):
  - `k6 run -e BASE_URL=http://localhost:8080 -e AUTH_TOKEN=Bearer_xxx perf/load/users.get.k6.js`
- Get user with CSRF (stress):
  - `k6 run -e BASE_URL=http://localhost:8080 -e AUTH_TOKEN=Bearer_xxx -e CSRF_TOKEN=abc123 perf/stress/users.get_with_csrf.k6.js`

Notes:
- Keep thresholds and stages inside each script to describe goals.
- Consider adding a `perf/.env.example` if many env vars are used.