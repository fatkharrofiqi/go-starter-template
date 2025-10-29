# Go Starter Backend Template

This is a **Go starter template** for building a backend service using **Gin** for HTTP routing, **JWT authentication**, **GORM with PostgreSQL**, and other essential tools like **Viper for configuration**, **Logrus for logging**, and **Migrate for database migrations**. It follows a clean architecture structure to keep the project modular and scalable.

---

## Features

- ✅ **JWT Authentication** (Login, Register, Refresh Token, Logout with Blacklist Token)
- ✅ **User Management Module**
- ✅ **Input Validation** using **Validator**
- ✅ **Configuration Management** with **Viper**
- ✅ **Structured Logging** with **Logrus**
- ✅ **Database ORM** using **GORM (PostgreSQL)**
- ✅ **Database Migrations** using **Migrate**
- ✅ **HTTP Routing** using **Fiber**
- ✅ **Middleware Support** for authentication
- ✅ **Monitoring** with **Jaeger** and **OpenTelemetry** for distributed tracing
- ✅ **Makefile** for easy project commands
- ✅ **Middleware Support** for authentication
- ✅ **YQ** for reading YAML configuration files
- ✅ **Unit testing** using **Testify** 
- ✅ **Performance testing** using **K6**
- ✅ **Redis Integration** for caching

---

## Authentication Flow

This application implements a secure JWT-based authentication system with cookie-based refresh token storage. Here's how it works:

### 🔐 Registration Flow
1. User submits registration data (email, password, etc.)
2. System validates input and creates new user account
3. User receives success response and can proceed to login

### 🔑 Login Flow
1. User submits login credentials (email/username and password)
2. System validates credentials against database
3. If valid, system generates:
   - **Access Token** (JWT) - Short-lived (15-30 minutes)
   - **Refresh Token** (JWT) - Long-lived (7-30 days)
4. Access token is returned in response body
5. **Refresh token is stored in HTTP-only cookie** for security
6. User can access protected routes using the access token

### 🔄 Token Refresh Flow
1. When access token expires, client receives 401 Unauthorized
2. Client automatically calls `/api/auth/refresh-token` endpoint
3. System reads refresh token from HTTP-only cookie
4. If refresh token is valid and not blacklisted:
   - Generate new access token
   - Optionally rotate refresh token (generate new one)
   - Return new access token in response
   - Store new refresh token in cookie (with refresh token rotation)
5. Client uses new access token for subsequent requests

### 🚪 Logout Flow
1. User calls `/api/auth/logout` endpoint
2. System adds current refresh token to blacklist
3. HTTP-only cookie containing refresh token is cleared
4. User is successfully logged out

### 🛡️ Security Features
- **HTTP-only cookies**: Refresh tokens stored in HTTP-only cookies prevent XSS attacks
- **Token blacklisting**: Logout functionality blacklists refresh tokens
- **Short-lived access tokens**: Minimizes exposure if access token is compromised
- **Secure cookie attributes**: Cookies use Secure and SameSite attributes
- **Token rotation**: Optional refresh token rotation for enhanced security

### 📱 Client Implementation Notes
- Access tokens should be stored in memory (not localStorage)
- Implement automatic token refresh on 401 responses
- Handle logout by clearing local access token and calling logout endpoint
- Cookies are handled automatically by browsers

---

## Project Structure

```
📦 project-root
 ┣ 📂 cmd                # Application entry point
 ┃ ┣ 📂 app
 ┃ ┃ ┗ 📜 main.go        # Main file
 ┃ ┗ 📂 seed
 ┃   ┗ 📜 main.go        # Seeder main file
 ┣ 📂 db/migration       # Database migrations
 ┃ ┣ 📜 000001_create_user.up.sql
 ┃ ┣ 📜 000001_create_user.down.sql
 ┃ ┣ 📜 000002_create_role_and_permission.down.sql
 ┃ ┗ 📜 000002_create_role_and_permission.up.sql
 ┣ 📂 internal           # Internal business logic
 ┃ ┣ 📂 config           # Configuration files
 ┃ ┃ ┣ 📂 env
 ┃ ┃ ┣ 📂 monitoring
 ┃ ┃ ┣ 📂 validation
 ┃ ┃ ┣ 📜 app.go
 ┃ ┃ ┣ 📜 fiber.go
 ┃ ┃ ┣ 📜 gorm.go
 ┃ ┃ ┣ 📜 logrus.go
 ┃ ┃ ┣ 📜 migration.go
 ┃ ┃ ┗ 📜 viper.go
 ┃ ┣ 📂 controller       # HTTP controllers
 ┃ ┃ ┣ 📜 auth_controller.go
 ┃ ┃ ┣ 📜 user_controller.go
 ┃ ┃ ┗ 📜 welcome_controller.go
 ┃ ┣ 📂 dto             # Data Transfer Objects
 ┃ ┃ ┣ 📂 converter     # Converter Data Transfer Objects
 ┃ ┃ ┣ 📜 auth_request.go
 ┃ ┃ ┗ 📜 auth_response.go
 ┃ ┣ 📂 middleware      # Middleware handlers
 ┃ ┃ ┣ 📜 auth_middleware.go
 ┃ ┃ ┗ 📜 cors_middleware.go
 ┃ ┣ 📂 model          # Database models
 ┃ ┃ ┗ 📜 user.go
 ┃ ┣ 📂 repository     # Database repositories
 ┃ ┃ ┣ 📜 repository.go
 ┃ ┃ ┗ 📜 user_repository.go
 ┃ ┣ 📂 route         # Routing setup
 ┃ ┃ ┗ 📜 route.go
 ┃ ┣ 📂 service       # Business logic
 ┃ ┃ ┗ 📜 auth_service.go
 ┃ ┣ 📂 utils         # Utility packages
 ┃ ┃ ┗ 📂 errcode
 ┣ 📂 test            # Testing
 ┃ ┣ 📂 performance   # K6 performance tests
 ┃ ┃ ┣ 📜 get-user.js
 ┃ 📜 config.example.yml
 ┃ 📜 config.yml
 ┣ 📜 go.mod         # Go module dependencies
 ┣ 📜 go.sum         # Go module checksum
 ┣ 📜 Makefile       # Makefile for running tasks
```

---

## Installation & Setup

### Prerequisites

- [Go](https://golang.org/dl/) (1.24+ recommended)
- [PostgreSQL](https://www.postgresql.org/)
- [Make](https://www.gnu.org/software/make/) (for running commands)
- [K6](https://k6.io/) (for performance testing)

### Install Dependencies

```sh
make install
```

### Install YQ

To install `yq`, use the following command:

```sh
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 && sudo chmod +x /usr/local/bin/yq
```

### Install K6

To install `k6` for performance testing:

```sh
# Linux/Ubuntu
sudo apt-get install k6

# macOS
brew install k6

# Windows (using Chocolatey)
choco install k6
```

### Environment Configuration

Rename `config.example.yml` to `config.yml` and configure your database settings.

### Run Migrations

```sh
make migrateup
```

### Start the Server

```sh
make run
```

Server will be available at `http://localhost:3000`.

---

## API Endpoints

### Auth Module

| Endpoint                 | Method | Description          | Auth Required |
|--------------------------|--------|----------------------|---------------|
| `/api/auth/register`     | POST   | Register new user    | No            |
| `/api/auth/login`        | POST   | Login user           | No            |
| `/api/auth/logout`       | POST   | Logout user          | Yes           |
| `/api/auth/refresh-token`| POST   | Refresh JWT token    | No*           |

*Requires valid refresh token in HTTP-only cookie

### User Module

| Endpoint          | Method | Description      | Auth Required |
|-------------------|--------|------------------|---------------|
| `/api/users/me`   | GET    | Get current user | Yes           |

### Request/Response Examples

#### Register
```bash
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

#### Login
```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### Access Protected Route
```bash
curl -X GET http://localhost:3000/api/users/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

#### Refresh Token
```bash
curl -X POST http://localhost:3000/api/auth/refresh-token \
  -H "Content-Type: application/json" \
  --cookie "refresh_token=YOUR_REFRESH_TOKEN"
```

---

## Testing

### Unit Testing

Run unit tests using:

```sh
go test ./...
```

### Performance Testing

Run performance tests using K6:

```sh
# Run specific performance test
k6 run test/performance/get-user.js
```

---

## Makefile Commands

| Command           | Description                  |
|-------------------|------------------------------|
| `make install`    | Install the dependencies     |
| `make run`        | Start the application        |
| `make migrateschema name=<schema_name>`  | Create new migration |
| `make migrateup`  | Apply database migrations    |
| `make migratedown`| Rollback database migrations |

---

## Contributing

Feel free to fork and modify this template to fit your needs. Pull requests are welcome!

---

## License

This project is licensed under the **MIT License**.

---

🚀 **Happy coding!**

## TODO / Roadmap

- [x] Fix JWT refresh secret usage: make `GetRefreshSecret()` return `jwt.refresh_secret` from config.
- [ ] Add unit tests for controllers (`auth_controller`, `user_controller`) and middleware (`auth_middleware`, `csrf_middleware`).
- [ ] Add integration tests using Fiber’s test utilities for auth flow (login, refresh, logout) and protected routes.
- [ ] Document CSRF usage and add client example for `GenerateCsrfToken` + protected POST flow.
- [ ] Implement optional refresh token rotation toggle in config and ensure old refresh tokens are blacklisted consistently.
- [x] Add rate limiting middleware (per IP) to `/api/auth/login`.
- [ ] Extend rate limiting to sensitive endpoints and consider per-user throttling.
- [ ] Introduce account lockout/backoff strategy on repeated failed logins.
- [ ] Add password reset flow (request, token, reset) and email placeholders.
- [ ] Extend user module with list sorting options (name, email, created_at) and validate bounds.
- [ ] Add OpenAPI/Swagger spec and publish under `/docs` for API discovery.
- [ ] Create Dockerfile and `docker-compose.yml` (Postgres + Redis) with `make compose-up` target.
- [ ] Set up CI (GitHub Actions) for `go test`, `golangci-lint`, security checks, and build.
- [ ] Add static analysis (`golangci-lint`) and formatting checks to Makefile.
- [ ] Strengthen error handling: unify `errcode` mapping, differentiate JWT invalid vs expired, and improve messages.
- [ ] Improve logging: attach request ID/correlation ID, include user UUID on protected routes, and standardize fields.
- [ ] Harden configuration: environment variable overrides, validation for critical fields (DB DSN, secrets).
- [ ] Monitoring/OTEL: make OTLP endpoint configurable per environment, add timeouts/retry, and document local Jaeger setup.
- [ ] Redis caching strategy: define keys, TTLs, and invalidation for user reads/updates; add tests.
- [ ] Database: add useful indexes (e.g., `users(email)`, `roles(name)`, `permissions(name)`), and constraints.
- [ ] Seeder: make operations idempotent (upsert) and parameterize sample data; add `make seed` docs.
- [ ] Security: configurable bcrypt cost, optional pepper, and audit logging for permission changes.
- [ ] Performance: review N+1 queries, ensure necessary `Preload` usage, and expand K6 tests/thresholds.
- [ ] Add pagination metadata examples in API docs and verify total pages logic in tests.
- [ ] Provide example client snippets (JS/TS) for login, token refresh, and authorized requests.