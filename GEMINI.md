# GoMusic Context

## Project Overview

**GoMusic** is a tool designed to migrate music playlists from Chinese music platforms (Netease Cloud Music, QQ Music, Qishui Music) to international platforms (Apple Music, YouTube Music, Spotify).

It consists of a **Golang** backend and a **Vue.js** frontend.

### Tech Stack

*   **Backend:** Golang (v1.23)
    *   **Web Framework:** Gin
    *   **ORM:** GORM (MySQL)
    *   **Cache:** Redis
    *   **Scraping/Parsing:** GoQuery, Otto (JavaScript runtime)
*   **Frontend:** Vue.js (v2/v3, check `package.json` to confirm, likely v3 based on `vue.config.js`)
    *   **UI Library:** ElementUI
    *   **HTTP Client:** Axios
*   **Infrastructure:** Docker Compose (MySQL, Redis)

## Directory Structure

*   `main.go`: Application entry point.
*   `handler/`: HTTP Request handlers and Router configuration (`router.go`).
*   `logic/`: Core business logic for fetching and parsing playlists (Netease, QQMusic, Qishui).
*   `repo/`: Data access layer.
    *   `db/`: MySQL interactions via GORM.
    *   `cache/`: Redis interactions.
*   `misc/`: Miscellaneous utilities.
    *   `models/`: Data models and constants.
    *   `log/`: Logging configuration.
    *   `utils/`: Helper functions (e.g., encryption).
*   `static/`: Frontend application source code.

## Development & Deployment

### Prerequisites

*   Go 1.23+
*   Node.js & Yarn
*   Docker & Docker Compose

### 1. Database Setup

Start MySQL and Redis using Docker Compose:

```bash
# Ensure docker-compose.yaml is present (refer to DEPLOYMENT.md)
docker compose up -d
```

**Configuration:**
*   **MySQL:** `repo/db/mysql.go` (Default: `go_music:12345678@tcp(127.0.0.1:3306)/go_music`)
*   **Redis:** `repo/cache/redis.go` (Default Port: `16379`, Password: `SzW7fh2Fs5d2ypwT`)

### 2. Backend Development

Run the backend server:

```bash
go mod tidy
go run main.go
```

The server typically runs on port `8081` (Check `misc/models/common.go` if uncertain).

### 3. Frontend Development

Navigate to the `static` directory:

```bash
cd static
yarn install
yarn serve
```

The frontend will run on `http://localhost:8080` (default).

**Configuration:**
*   Update backend API URL in `static/src/App.vue` if necessary (defaults to `http://127.0.0.1:8081` for local dev).

### 4. Build

**Backend:**
```bash
go build -o GoMusic
```

**Frontend:**
```bash
cd static
yarn build
```

## Conventions

*   **Architecture:** Follows a Layered Architecture: `Handler` -> `Logic` -> `Repo`.
*   **Error Handling:** Use `log.Errorf` for logging errors.
*   **Database:** GORM is used for DB operations. `AutoMigrate` is used in `init()` for schema updates.
*   **Testing:** Unit tests are located alongside source files (e.g., `logic/neteasy_test.go`).
