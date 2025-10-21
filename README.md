# 🚀 Realtime Log Analytics (Go + Gin)

**Realtime Log Analytics** es una plataforma escrita 100% en **Go** que permite **recibir, procesar, almacenar y visualizar logs en tiempo real**.  
Diseñada como base para construir un **SaaS de observabilidad**, similar a Datadog o Logtail, pero mucho más ligero, rápido y portable.

---

## 🧭 Descripción general

El sistema recibe logs JSON mediante HTTP (`POST /ingest`), los procesa de forma concurrente con goroutines, los almacena en **PostgreSQL**, y los distribuye en tiempo real a los clientes conectados por **WebSocket** (`/ws`).

Está pensado para desarrolladores que quieren aprender Go en un contexto **real, escalable y monetizable**.

---

## 🧱 Arquitectura general

```
┌──────────────────────────────┐
│        Cliente/App           │
│  → POST /ingest (JSON logs)  │
└──────────────┬───────────────┘
               │
┌──────────────▼───────────────┐
│        API Server (Go)       │
│ - Validación API Key         │
│ - Envío a canal interno      │
└──────────────┬───────────────┘
               ▼
┌──────────────────────────────┐
│    Log Processor (Workers)   │
│ - Concurrencia con goroutines│
│ - Enriquecimiento y guardado │
│ - Stream a WebSocket clients │
└──────────────┬───────────────┘
               ▼
┌──────────────────────────────┐
│          PostgreSQL          │
│ - Almacenamiento persistente │
└──────────────────────────────┘
```

---

## 🧩 Estructura del proyecto

```
realtime-logs/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── api/
│   │   ├── ingest.go
│   │   └── query.go
│   ├── auth/
│   │   └── api_keys.go
│   ├── db/
│   │   └── database.go
│   ├── models/
│   │   └── log.go
│   └── stream/
│       └── websocket.go
├── go.mod
├── Dockerfile
├── .env
└── README.md
```

---

## 🛠️ Instalación y ejecución local

```bash
git clone https://github.com/tuusuario/realtime-logs.git
cd realtime-logs
go mod tidy
go run cmd/server/main.go
```

Verifica: [http://localhost:8080/health](http://localhost:8080/health)

---

## 🧪 Prueba rápida de ingesta

```bash
curl -X POST http://localhost:8080/ingest -H "X-API-Key: test123" -H "Content-Type: application/json" -d '{"org_id":"acme","level":"info","message":"User logged in"}'
```

---

## 💰 Monetización (SaaS)

1. **Freemium:** límite de ingestión (1 GB/mes)  
2. **Pro:** pago mensual por volumen o retención  
3. **Enterprise:** dashboards y alertas personalizadas  

Integrable con Stripe para gestión de pagos.

---

## 🧾 Próximos pasos

| Paso | Descripción | Resultado esperado |
|------|--------------|--------------------|
| 1️⃣ | Conexión DB (GORM) | Servidor conectado |
| 2️⃣ | Middleware API key | Seguridad básica |
| 3️⃣ | Endpoint `/ingest` | Logs almacenados |
| 4️⃣ | WebSocket `/ws` | Logs en tiempo real |
| 5️⃣ | `/query` | Filtros de logs |
| 6️⃣ | Docker Compose | Entorno reproducible |
| 7️⃣ | Dashboard (React) | Interfaz visual |
| 8️⃣ | Stripe | Monetización lista |

---

MIT License — libre para usar y modificar.
