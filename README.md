# Форма обратной связи для нахождения багов

## Build & Deploy

Этот файл содержит инструкции по сборке и деплою.

## Требования

- Docker + Docker Compose
- (опционально) Node.js 18+ для локальной сборки фронта
- (опционально) Go 1.22+ для локальных go-команд

## Переменные окружения

Создать `.env` на сервере/локально по шаблону `.env.example`.

## Deploy

Из корня репозитория:

```bash
docker compose --env-file .env -f deployments/docker-compose.yml up -d --build
```

Проверка:

- API: `http://localhost:8080/healthz`
- Swagger UI (если включён): `http://localhost:8081`

Остановить:

```bash
docker compose --env-file .env -f deployments/docker-compose.yml down
```

## Модераторы

Модераторы сидятся при старте API из env.

Поддерживаются переменные `MOD_SEED_1_*` … `MOD_SEED_5_*`:

- `MOD_SEED_N_EMAIL`
- `MOD_SEED_N_NAME`
- `MOD_SEED_N_PASSWORD`

## Build (локально)

### Backend

```bash
go test ./...
go build ./cmd/api
```

### Frontend

```bash
cd frontend
npm ci
npm run build
```
