<p align="center">
  <img src="brand/renders/readme-header.png" alt="fai" width="1280">
</p>

<p align="center">
  <a href="https://github.com/alialjaffer/fai/actions/workflows/ci.yml"><img src="https://github.com/alialjaffer/fai/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
</p>

# fai

Self-hosted workspace for tasks, notes, reminders, and calendars. Run it as a
web app, Docker Compose stack, or desktop app.

Features include task boards and lists, Markdown notes, calendars, time
tracking, reminders, Jira, optional AI, and themes. See the
[installation guide](docs/INSTALLATION_GUIDE.md) for feature details and setup.

## Install

### Homebrew

```sh
brew install --cask alialjaffer/tap/fai
```

### Docker Compose

```sh
cp .env.prod.example .env.prod
make hash-password
# Set the domain, auth values, and calendar token in .env.prod.
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### Desktop

Download an installer from the [latest release](https://github.com/alialjaffer/fai/releases/latest). To build locally:

```sh
make electron-dist
```

### Local development

Requirements: Go 1.27, Node 22, Docker, and SQLite or PostgreSQL.

```sh
make dev-db-up
make migrate-up
cp backend/.env.example backend/.env
make backend-run
make frontend-run
```

See the [installation guide](docs/INSTALLATION_GUIDE.md) for full setup,
configuration, and integrations.

## Screenshots

All screenshots use synthetic tasks, notes, and labels.

| Board | List |
|:---:|:---:|
| ![Board](screenshots/board.png) | ![List](screenshots/list.png) |

| Stats | Calendar |
|:---:|:---:|
| ![Stats](screenshots/stats.png) | ![Calendar](screenshots/calendar.png) |

## Stack

Go, React, Vite, Electron, SQLite, and PostgreSQL.

## Links

- [Installation guide](docs/INSTALLATION_GUIDE.md)
- [Security policy](SECURITY.md)
- [License](LICENSE)
