# Registro de cambios — CSV Copilot

Este archivo registra las modificaciones del proyecto por etapa. Se actualizará a medida que avance el trabajo.

## Etapa 1 — Backend y carga de CSV

- Se conectaron `GET /health`, `POST /api/datasets` y `GET /api/datasets/{id}` al servidor HTTP.
- Se implementó lectura y validación CSV con `encoding/csv`: encabezados, filas consistentes, conteo, tipos básicos y vista previa de hasta cinco registros.
- Se agregó un repository en memoria con identificadores aleatorios para guardar y recuperar datasets.
- La carga multipart limita el archivo a 10 MB y devuelve errores HTTP para archivos vacíos o inválidos y datasets inexistentes.
- Se actualizó el módulo a Go 1.26 y se añadieron pruebas de servicio, repository y endpoints.
- Verificación realizada: `gofmt` y `go test ./...` pasaron.

## Etapa 2 — Motor de análisis CSV

- Se creó `pkg/analysis/operations.go` con `count_rows`, `sum`, `average`, `min`, `max` y `group_by` con agregaciones `count` y `sum`.
- Los cálculos validan columnas y tipos numéricos. Las celdas numéricas vacías se omiten; los valores no numéricos inválidos generan errores. `average`, `min` y `max` informan cuando no hay valores numéricos.
- Se creó `internal/service/analysis_service.go` para recuperar el dataset por ID mediante el repository e invocar el motor. Se usa la interfaz `DatasetProvider` para evitar ciclos de dependencias.
- Se añadieron pruebas unitarias de operaciones y servicio, incluidas validaciones y agrupaciones.
- Se corrigió `cmd/demo/main.go` para sumar `total_venta` por región.
- Verificación realizada: `gofmt`, `go test ./...` y `go run ./cmd/demo` pasaron.

## Etapa 3 — Integración con Ollama

- Se centralizaron `OLLAMA_URL`, `OLLAMA_MODEL` y `OLLAMA_TIMEOUT_SECONDS` en `config`, con valores predeterminados `http://localhost:11434`, `qwen3:4b` y 60 segundos.
- Se implementó `internal/service/llm_service.go` con `net/http` y `encoding/json`, `/api/chat`, streaming deshabilitado, temperatura cero y esquema JSON restringido a las operaciones disponibles más el marcador de pregunta no interpretable.
- El modelo recibe la pregunta, los nombres y tipos de columna y las operaciones permitidas; no recibe filas ni contenido completo del CSV.
- Se validan estrictamente operación, parámetros, existencia de columnas y compatibilidad numérica antes de invocar `AnalysisService`.
- Se agregó `POST /api/datasets/{id}/query`; devuelve pregunta, instrucción validada y resultado, sin exponer la respuesta cruda del modelo.
- Se añadieron pruebas con `httptest` para configuración, solicitudes a Ollama, timeout, respuestas e instrucciones inválidas y el recorrido completo del endpoint.
- Se actualizó `README.md` con configuración, uso del endpoint y comando cURL.
- Verificación realizada: `gofmt` y `go test ./...` pasaron. Las pruebas de Ollama usan un servidor mock y no requieren una instancia real.
- Se creó `EXPLICACIONINCORPORACION.MD` con el flujo Qwen-Ollama-Go, configuración, validaciones, ejecución manual, errores y pruebas.

## Etapa 4 — Integración y estabilización

- Se agregó `internal/handlers/dataset_query_integration_test.go`: carga CSV multipart, consulta de metadatos y siete operaciones usando el mismo ID, con Ollama simulado.
- El recorrido automatizado verifica `count_rows`, `sum`, `average`, `min`, `max`, `group_by/count` y `group_by/sum`, incluyendo sumas agrupadas en cero.
- Se añadieron verificaciones HTTP para CSV vacío, filas mal formadas, exceso del límite, IDs inexistentes, operación o parámetros inválidos, columnas inexistentes/no numéricas, pregunta ambigua, respuesta JSON inválida, desconexión y timeout de Ollama.
- Los errores de interpretación devuelven mensajes genéricos comprensibles sin incluir detalles internos del transporte o parser. El handler de carga elimina archivos temporales de multipart al terminar.
- Se agregó una prueba concurrente para `Save`/`Get`; el repository protege el mapa con `RWMutex`. El modelo recibe solamente pregunta, encabezados y tipos, y su salida solo puede seleccionar operaciones predefinidas.
- Se actualizó `README.md` con endpoints, configuración, ejecución, pruebas y una demostración completa en PowerShell.
- Verificación automatizada: `gofmt`, `go test ./...` y `go build ./...` completaron correctamente.
- Se añadió y ejecutó una prueba concurrente de `Save`/`Get`. También se intentó `go test -race ./...`, pero el entorno tiene `CGO_ENABLED=0`; Go requiere CGo para activar el detector de carreras, por lo que ese detector no pudo ejecutarse.
- Verificación manual con Ollama real y `qwen3:4b`: carga de 41 filas, lectura de metadatos y consulta `group_by/sum` por vendedor con resultados Ana 4803000, Bruno 3335000, Carla 4115000 y Diego 4071000. Para la prueba se usó el puerto 18080.
- No quedaron tareas pendientes de la etapa 4; la interfaz web se incorporó en la etapa 5.

## Etapa 5 - Interfaz web

- Se crearon `web/index.html`, `web/style.css` y `web/app.js` con HTML, CSS y JavaScript vanilla en tema oscuro responsive y acento `#7500F5`.
- La interfaz permite seleccionar, arrastrar y soltar CSV, muestra nombre/tamaño, valida contenido y límite de 10 MB, muestra carga y permite reemplazar el dataset.
- Se conecta al backend para comprobar `/health`, subir con `POST /api/datasets`, recuperar metadatos con `GET /api/datasets/{id}` y consultar con `POST /api/datasets/{id}/query`.
- Muestra columnas, tipos, vista previa, operación y parámetros interpretados, valores y agrupaciones recibidos. No calcula resultados en JavaScript.
- El backend sirve `web/` en `GET /`; la UI y la API comparten origen, así que no hizo falta configurar CORS.
- La UI impide comenzar una carga y una consulta simultáneamente. La documentación aclara que el MVP no contempla procesamiento paralelo ni garantiza accesos concurrentes al repository en memoria.
- Verificación realizada: `node --check web/app.js`, `go test ./...`, `go build ./...` y solicitudes HTTP locales a `/`, `/style.css` y `/app.js` (200). La inspección visual asistida del navegador se detuvo porque Computer Use no pudo determinar la URL actual con suficiente confianza; los breakpoints responsive quedaron implementados, pero la revisión visual manual de tamaños distintos queda pendiente.

## Etapa 6 — Valores únicos y filtros básicos

- Se incorporaron `count_distinct` (ignora vacíos, recorta espacios y no distingue mayúsculas/minúsculas) y `count_filtered` en el motor y `AnalysisService`.
- `count_filtered` admite `eq`, `neq`, `contains`, `gt`, `gte`, `lt` y `lte`, con validación de columnas, tipos y valor; cada consulta acepta una sola condición.
- Se amplió el esquema y el prompt de Ollama para distinguir conteo de filas, valores únicos y registros filtrados, con columnas/tipos disponibles y operadores permitidos.
- La UI muestra los parámetros nuevos y presenta los resultados numéricos como valores escalares; no realiza cálculos en JavaScript.
- Se añadieron pruebas unitarias e integración HTTP con Ollama simulado para las nuevas operaciones.
- Se corrigió la validación de `count_filtered` para normalizar cadenas numéricas válidas devueltas por algunos modelos locales a números antes de ejecutar el filtro; valores no numéricos siguen rechazándose. El prompt ahora especifica el tipo JSON esperado según la columna.
- Se reforzó la instrucción de `count_distinct` con un ejemplo explícito para “¿Cuántos vendedores hay?” y el uso exacto del encabezado `vendedor`; el recorrido HTTP simulado cubre ahora esa pregunta en español.
- Siguen sin implementarse filtros combinados/avanzados ni SQL. El MVP no contempla procesamiento paralelo y no se debe asumir seguridad del repository en memoria ante accesos concurrentes.
