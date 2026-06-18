# Performance benchmark

Benchmark local de ingesta HTTP sobre el store en memoria.

## Entorno

- Fecha: 2026-06-18
- Hardware: AMD Ryzen 7 PRO 5850U with Radeon Graphics
- CPU: 16 hilos
- Memoria: 14 GiB
- Go: 1.24.4
- Store: memoria, capacidad actual 10,000 logs retenidos
- Servidor:

```bash
GIN_MODE=release \
ACCESS_LOGS=false \
INGEST_WORKERS=8 \
INGEST_QUEUE_SIZE=65536 \
PORT=8080 \
./realtime-logs-server
```

## Metodologia

La prueba usa `tools/loadtest`, que envia requests HTTP `POST /ingest` con un log por request y mide throughput, exitos/fallos y latencias del lado cliente.

Antes de las mediciones se ejecuto un warmup de 1,000 requests con concurrencia 20. Las rutas se probaron sin API key y sin PostgreSQL para aislar la capacidad de la API, parser JSON, cola asincrona, workers y store en memoria.

## Resultados

| Requests | Concurrencia | Duracion | Exitos | Fallos | RPS | p50 | p90 | p99 | Max |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 5,000 | 1 | 1.24s | 5,000 | 0 | 4,030.8 | 0.2 ms | 0.3 ms | 0.3 ms | 1.0 ms |
| 10,000 | 10 | 0.77s | 10,000 | 0 | 13,026.1 | 0.7 ms | 1.0 ms | 1.4 ms | 2.7 ms |
| 20,000 | 50 | 1.42s | 20,000 | 0 | 14,125.3 | 2.6 ms | 6.1 ms | 12.0 ms | 23.8 ms |
| 30,000 | 100 | 1.58s | 30,000 | 0 | 19,037.6 | 3.2 ms | 7.3 ms | 31.4 ms | 93.8 ms |
| 30,000 | 250 | 1.96s | 30,000 | 0 | 15,274.9 | 8.0 ms | 24.6 ms | 115.8 ms | 436.9 ms |
| 30,000 | 500 | 1.99s | 30,000 | 0 | 15,045.9 | 17.3 ms | 57.7 ms | 194.1 ms | 390.0 ms |

## Lectura

- Pico observado: 19,037.6 requests/segundo con 100 clientes concurrentes.
- Todas las corridas completaron con 0 fallos.
- La latencia p99 se mantuvo en 31.4 ms en el mejor punto de throughput.
- A partir de 250 clientes concurrentes el throughput baja y la p99 sube, senal de saturacion por contencion/planificacion.
- La consulta posterior devolvio `total=10000`, consistente con la capacidad FIFO configurada del store en memoria.
- RSS observado del proceso despues de la prueba: ~46 MiB.

## Frases para CV

- Built a Go/Gin realtime log ingestion API with async workers, WebSocket streaming, API-key auth, and memory/PostgreSQL storage.
- Benchmarked the ingestion path at 19k+ HTTP log events/sec on local hardware with 0 failed requests and p99 latency near 31 ms at 100 concurrent clients.
- Added tunable ingestion workers, queue backpressure, configurable access logs, and focused API/store tests.

## Limites de la medicion

- Es un benchmark local, no distribuido.
- La prueba usa store en memoria; PostgreSQL tendra otros limites por I/O, indices y configuracion.
- Cada request contiene un solo log; batches grandes pueden mejorar throughput por overhead HTTP menor.
- No mide streaming WebSocket con clientes conectados.
