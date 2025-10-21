# Realtime Logs – Documentación detallada

Esta guía cubre instalación, configuración, API, modelo de datos, arquitectura y resolución de problemas.

## 1. Resumen
- API HTTP para ingesta y consulta de logs
- Pipeline concurrente con goroutines
- WebSocket para streaming en tiempo real
- Almacenamiento en memoria y soporte opcional PostgreSQL (GORM)
- Autenticación simple por API key

## 2. Instalación y ejecución
### Requisitos
- Go 1.23+ (herramienta usa toolchain 1.24.4)
- (Opcional) PostgreSQL 14+

### Variables de entorno
- PORT: puerto HTTP (default 8080)
- API_KEYS: lista de claves separadas por coma; activa el middleware X-API-Key (vacío = desactivado)
- POSTGRES_DSN: DSN de PostgreSQL; si está presente se usa Postgres, si no memoria

Puedes copiar `.env.example` a `.env` y editar.

### Ejecutar (PowerShell)
```
$env:PORT = "9090"
# $env:API_KEYS = "dev-key"
# $env:POSTGRES_DSN = "host=localhost user=postgres password=postgres dbname=realtime_logs port=5432 sslmode=disable TimeZone=UTC"

go run ./cmd/server
```

Prueba salud:
```
iwr http://localhost:9090/health
```

## 3. API
### Ingesta
- POST /ingest y POST /api/ingest
- Headers: Content-Type: application/json; X-API-Key si corresponde
- Cuerpos aceptados:
  - Array: { "items": [ {"org_id":"acme","level":"info","message":"hi","timestamp":"2025-10-15T22:00:00Z"}, ... ] }
  - Objeto: { "org_id":"acme","level":"info","message":"one" }
- Respuesta: 202 { "enqueued": N }

### Consulta
- GET /query y GET /api/query
- Params: org_id, level, q, from, to (RFC3339), limit, offset
- Respuesta: { total, items }

### WebSocket
- GET /ws (protegido por X-API-Key si `API_KEYS` configurado)
- Emite arrays JSON con nuevos logs ingeridos

## 4. Modelo de datos
```
type LogEntry struct {
    ID        uint      `gorm:"primaryKey"`
    OrgID     string    `json:"org_id" binding:"required"`
    Level     string    `json:"level" binding:"required"`
    Message   string    `json:"message" binding:"required"`
    Timestamp string    `json:"timestamp"`
    Ts        time.Time `json:"-" gorm:"index"`
}
```
- Timestamp se normaliza a RFC3339 si falta
- Ts se calcula al ingerir para ordenar/filtrar

## 5. Arquitectura
- `cmd/server/main.go`: arranque; carga .env; wiring de rutas; selección de store; workers; WS
- `internal/api/ingest.go`: HTTP -> cola -> workers -> Store + WS
- `internal/api/query.go`: filtros/paginación contra Store
- `internal/auth/api_keys.go`: middleware X-API-Key
- `internal/db/database.go`: MemoryStore y PostgresStore
- `internal/stream/websocket.go`: Hub WS (upgrade, registro, broadcast)

## 6. Persistencia
- Memoria: slice con RWMutex y capacidad fija (recorte FIFO)
- PostgreSQL: GORM, AutoMigrate, consultas con filtros y orden por Ts DESC

## 7. Seguridad
- Define `API_KEYS` y envía `X-API-Key` en clientes HTTP/WS

## 8. Docker
```
docker build -t realtime-logs .
docker run --rm -p 8080:8080 realtime-logs
# Con vars
docker run --rm -e PORT=9090 -e API_KEYS="key1,key2" -p 9090:9090 realtime-logs
# Con Postgres
docker run --rm -e POSTGRES_DSN="host=host.docker.internal user=postgres password=postgres dbname=realtime_logs port=5432 sslmode=disable TimeZone=UTC" -p 8080:8080 realtime-logs
```

## 9. Troubleshooting
- Puerto ocupado → cambia PORT
- 401/403 → define API_KEYS y envía header
- Sin resultados → ingiere primero y consulta filtros correctos
- Postgres → revisa DSN y conectividad; el esquema migra automáticamente

## 10. Próximos pasos
- docker-compose con Postgres
- Índices y filtros avanzados
- UI de dashboard
- Cuotas, multitenancy, alertas

## 11. Docker Compose

El archivo `docker-compose.yml` incluye dos servicios:
- `db`: PostgreSQL 14 (usuario/password por defecto postgres)
- `app`: la aplicación Go, expuesta en `http://localhost:8080`

Para levantar todo:
```
docker compose up --build
```

Luego prueba:
```
iwr http://localhost:8080/health
```

Notas:
- `depends_on` espera a que la base esté saludable antes de arrancar `app`.
- La app recibe `POSTGRES_DSN` para usar Postgres en lugar de memoria.
