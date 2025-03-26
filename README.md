# Go Starter Backend Template

This is a **Go starter template** for building a backend service using **Gin** for HTTP routing, **JWT authentication**, **GORM with PostgreSQL**, and other essential tools like **Viper for configuration**, **Logrus for logging**, and **Migrate for database migrations**. It follows a clean architecture structure to keep the project modular and scalable.

---

## Features

- ✅ **JWT Authentication** (Login, Register, Refresh Token)
- ✅ **User Management Module**
- ✅ **Input Validation** using **Validator**
- ✅ **Configuration Management** with **Viper**
- ✅ **Structured Logging** with **Logrus**
- ✅ **Database ORM** using **GORM (PostgreSQL)**
- ✅ **Database Migrations** using **Migrate**
- ✅ **HTTP Routing** using **Fiber**
- ✅ **Makefile** for easy project commands
- ✅ **Middleware Support** for authentication
- ✅ **YQ** for reading YAML configuration files
- ✅ **Unit testing** using **Testify** 

---

## Project Structure

```
📦 project-root
 ┣ 📂 cmd                # Application entry point
 ┃ ┗ 📜 main.go          # Main file
 ┣ 📂 db/migration       # Database migrations
 ┃ ┣ 📜 000001_create_user.up.sql
 ┃ ┗ 📜 000001_create_user.down.sql
 ┣ 📂 internal           # Internal business logic
 ┃ ┣ 📂 config           # Configuration files
 ┃ ┃ ┣ 📂 validation
 ┃ ┃ ┣ 📜 app.go
 ┃ ┃ ┣ 📜 constant.go
 ┃ ┃ ┣ 📜 fiber.go
 ┃ ┃ ┣ 📜 gorm.go
 ┃ ┃ ┣ 📜 logrus.go
 ┃ ┃ ┣ 📜 migration.go
 ┃ ┃ ┣ 📜 validator.go
 ┃ ┃ ┗ 📜 viper.go
 ┃ ┣ 📂 controller       # HTTP controllers
 ┃ ┃ ┣ 📜 auth_controller.go
 ┃ ┃ ┗ 📜 user_controller.go
 ┃ ┣ 📂 dto             # Data Transfer Objects
 ┃ ┃ ┣ 📜 auth_request.go
 ┃ ┃ ┗ 📜 auth_response.go
 ┃ ┣ 📂 middleware      # Middleware handlers
 ┃ ┃ ┗ 📜 auth_middleware.go
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
 ┃ ┃ ┣ 📂 jwtutil
 ┃ ┃ ┗ 📂 logutil
 ┣ 📂 test            # Testing
 ┃ 📜 config.example.yml
 ┃ 📜 config.yml
 ┣ 📜 go.mod         # Go module dependencies
 ┣ 📜 go.sum         # Go module checksum
 ┣ 📜 Makefile       # Makefile for running tasks
```

---

## Installation & Setup

### Prerequisites

- [Go](https://golang.org/dl/) (1.19+ recommended)
- [PostgreSQL](https://www.postgresql.org/)
- [Make](https://www.gnu.org/software/make/) (for running commands)

### Install Dependencies

```sh
make install
```

### Install YQ

To install `yq`, use the following command:

```sh
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 && sudo chmod +x /usr/local/bin/yq
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

| Endpoint                 | Method | Description          |
|--------------------------|--------|----------------------|
| `/api/auth/register`     | POST   | Register new user    |
| `/api/auth/login`        | POST   | Login user           |
| `/api/auth/refresh-token`| POST   | Refresh JWT token    |

### User Module

| Endpoint          | Method | Description      |
|-------------------|--------|------------------|
| `/api/users/me`   | GET    | Get current user |

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
