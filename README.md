# Realtime Log Analytics

API ligera en Go para recibir, consultar y transmitir logs en tiempo real. El servicio acepta eventos JSON por HTTP, los procesa con workers, los guarda en memoria o PostgreSQL y publica nuevos batches a clientes WebSocket.

## Capacidades

- Ingesta HTTP: `POST /ingest` y `POST /api/ingest`
- Consulta: `GET /query` y `GET /api/query`
- Streaming en tiempo real: `GET /ws`
- Autenticacion opcional por `X-API-Key`
- Persistencia en memoria por defecto o PostgreSQL con `POSTGRES_DSN`
- Workers y cola de ingesta configurables
- Entorno reproducible con Docker Compose

## Estructura

```text
realtime-logs/
├── cmd/server/              # Entrada de la API
├── internal/api/            # Handlers HTTP
├── internal/auth/           # Middleware de API keys
├── internal/db/             # Stores memoria/PostgreSQL
├── internal/models/         # Modelo LogEntry
├── internal/stream/         # WebSocket hub
├── tools/loadtest/          # Cliente de carga simple
├── docs/                    # Documentacion extendida
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

## Ejecutar localmente

```bash
go mod download
go run ./cmd/server
```

Verifica el servicio:

```bash
curl http://localhost:8080/health
```

Por defecto no hay autenticacion si `API_KEYS` esta vacio. Para activarla:

```bash
API_KEYS=dev-key go run ./cmd/server
```

Para benchmarks o cargas locales de alto volumen:

```bash
ACCESS_LOGS=false INGEST_WORKERS=8 INGEST_QUEUE_SIZE=20000 go run ./cmd/server
```

## Probar ingesta y consulta

Sin API key configurada:

```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"org_id":"acme","level":"info","message":"User logged in"}'

curl "http://localhost:8080/query?org_id=acme&limit=10"
```

Con `API_KEYS=dev-key`:

```bash
curl -X POST http://localhost:8080/ingest \
  -H "X-API-Key: dev-key" \
  -H "Content-Type: application/json" \
  -d '{"org_id":"acme","level":"error","message":"Payment failed"}'
```

## Docker Compose

Levanta API y PostgreSQL:

```bash
docker compose up --build
```

Compose configura `API_KEYS=dev-key`, por lo que las rutas protegidas requieren:

```bash
curl http://localhost:8080/health
curl -H "X-API-Key: dev-key" "http://localhost:8080/api/query?limit=10"
```

## Documentacion

- [Guia tecnica](docs/technical.md)
- [Benchmark de rendimiento](docs/performance.md)
- [Evaluacion del proyecto](docs/evaluation.md)
- [Potencial de negocio](docs/business-potential.md)

## Herramienta de carga

```bash
go run ./tools/loadtest -url http://localhost:8080/ingest -api-key dev-key -n 500 -c 25
```

Si no configuraste `API_KEYS`, pasa `-api-key ""`.

## Licencia

MIT.
