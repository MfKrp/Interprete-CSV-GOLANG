# Integración de Qwen, Ollama y Go

Este documento da una explicación de como es la incorporación de QWEN:4b junto con GO, para el proyecto a presentar para el curso de incorporación de lenguajes con GO dictado por la SADIO durante el mes de octubre del año 2026.

## Resumen

CSV Copilot usa tres componentes con responsabilidades distintas:

- **Qwen (`qwen3:4b`)** interpreta la pregunta y propone una operación y sus parámetros.
- **Ollama** ejecuta Qwen localmente y ofrece una API HTTP para recibir mensajes y devolver su respuesta.
- **Go** recibe la pregunta, envía el esquema del CSV a Ollama, valida la instrucción estructurada y ejecuta el cálculo con el motor del proyecto.

Qwen no calcula los totales ni recibe el archivo completo. Solo propone cuál de las operaciones existentes representa la pregunta. Go mantiene el control de los datos y realiza los cálculos.

## Preparación y configuración

Instalá Ollama y descargá el modelo:

```powershell
ollama pull qwen3:4b
```

Ollama expone su API local en `http://localhost:11434` por defecto. `config/config.go` centraliza estos valores:

| Variable | Valor predeterminado | Uso |
| --- | --- | --- |
| `OLLAMA_URL` | `http://localhost:11434` | URL base del servidor Ollama. |
| `OLLAMA_MODEL` | `qwen3:4b` | Modelo que interpreta las consultas. |
| `OLLAMA_TIMEOUT_SECONDS` | `60` | Timeout HTTP; rango permitido de 1 a 300 segundos. |
| `PORT` | `8080` | Puerto del backend Go. |

`.env.example` muestra las variables. El proceso Go lee variables de entorno; si usás un archivo `.env`, cargalo en el entorno antes de iniciar la aplicación.

## Componentes

### Inicio del servidor

`cmd/app/main.go` carga configuración, crea un `DatasetRepository` en memoria y registra los handlers HTTP. También crea:

1. `AnalysisService`, que delega las operaciones en el motor existente.
2. `OllamaClient`, configurado con la URL, el modelo y un `http.Client` con timeout.
3. `QueryHandler`, que conecta la interpretación con la ejecución.

### Cliente de Ollama

`internal/service/llm_service.go` envía una solicitud con `net/http` y `encoding/json` a `POST {OLLAMA_URL}/api/chat`. Incluye el modelo, la pregunta, los nombres y tipos de columnas y las operaciones permitidas. No incluye filas, vista previa ni contenido completo del CSV.

La solicitud desactiva streaming con `stream: false`, pide `think: false`, fija temperatura cero y usa `format` con JSON Schema. El esquema restringe la salida a las operaciones del motor y los campos que requieren. Ollama documenta tanto la respuesta no transmitida en streaming como la salida restringida por JSON Schema en [`/api/chat`](https://docs.ollama.com/api/chat).

### Validación y ejecución

Ollama devuelve el texto JSON dentro de `message.content`. Go lo decodifica con campos estrictos y lo convierte a `AnalysisRequest`. Antes de ejecutar, Go comprueba:

- Que la operación sea una de las permitidas: `count_rows`, `count_distinct`, `count_filtered`, `sum`, `average`, `min`, `max` o `group_by`.
- Que sus parámetros requeridos estén presentes y los no aplicables no se hayan agregado.
- Que las columnas existan en el esquema del dataset.
- Que los cálculos matemáticos usen columnas de tipo `number`.
- Que `group_by` use `count` o `sum`; la agregación `sum` necesita una columna objetivo numérica.

Si la pregunta es ambigua, el modelo puede devolver el marcador `uninterpretable`; Go lo rechaza y no ejecuta nada. La salida del modelo nunca se trata como código, SQL o comandos.

`AnalysisService` recupera las filas por ID desde el repository y llama a `pkg/analysis/operations.go`. Una suma de ventas por vendedor, por ejemplo, se representa así:

```json
{
  "operation": "group_by",
  "group_column": "vendedor",
  "aggregation": "sum",
  "target_column": "total_venta"
}
```

Go convierte esos campos a `AnalysisRequest` (`GroupBy`, `Aggregation` y `SumColumn`) y el motor suma los valores de las filas almacenadas.

## Flujo completo

1. El cliente carga el CSV mediante `POST /api/datasets`.
2. Go valida el archivo y lo guarda en memoria bajo un ID.
3. El cliente envía una pregunta a `POST /api/datasets/{id}/query`.
4. `QueryHandler` recupera las columnas y tipos del dataset.
5. `OllamaClient` envía la pregunta y el esquema a Qwen a través de Ollama.
6. Go valida la operación estructurada antes de ejecutarla.
7. `AnalysisService` recupera los registros y calcula el resultado.
8. El endpoint devuelve la pregunta, la instrucción validada y el resultado; no devuelve el texto crudo de Qwen.

```text
Cliente HTTP -> QueryHandler -> DatasetRepository (columnas y tipos)
                      |
                      +-> Ollama -> Qwen
                      |    pregunta + esquema -> instrucción JSON
                      |
                      +-> validación -> AnalysisService -> pkg/analysis
                                                      -> filas -> resultado JSON
```

## Ejecución manual

Con Ollama activo y `qwen3:4b` descargado, iniciá el backend:

```powershell
go run ./cmd/app
```

En otra terminal, cargá el CSV y copiá el `id` devuelto:

```powershell
curl.exe -F "file=@ventas_demo.csv" http://localhost:8080/api/datasets
```

Luego consultá usando ese ID:

```powershell
$id = "ID_DEVUELTO_POR_LA_CARGA"
$body = @{ question = "Sumá la columna total_venta agrupada por vendedor." } | ConvertTo-Json -Compress
$response = Invoke-RestMethod -Method Post `
  -Uri "http://localhost:8080/api/datasets/$id/query" `
  -ContentType "application/json; charset=utf-8" `
  -Body ([Text.Encoding]::UTF8.GetBytes($body))
$response | ConvertTo-Json -Depth 8
```

También podés probar preguntas como:

- `¿Cuántas filas tiene este archivo?`
- `¿Cuál es el promedio de unidades?`
- `¿Cuál fue la venta máxima?`
- `¿Cuántas ventas hubo por región?`

Para una suma monetaria, nombrá explícitamente `total_venta`; una pregunta como “cuánto vendió” puede ser interpretada como suma de unidades.

## Conteos distintos y filtros básicos

`count_distinct` cuenta valores no vacíos tras recortar espacios externos y comparar sin distinguir mayúsculas/minúsculas. `count_filtered` cuenta filas con una condición sobre una columna. Admite `eq`, `neq`, `contains`, `gt`, `gte`, `lt` y `lte`: `eq`/`neq` aceptan valores del tipo de la columna, `contains` solo texto, y las comparaciones de orden solo números. Go valida columna, operador y tipo de valor antes de evaluar las filas.

Ejemplos: “¿Cuántos vendedores hay?”, “¿Cuántos registros tienen región Norte?” y “¿Cuántas ventas superan los 10000?”. Cada consulta filtrada admite una sola condición. No se implementan filtros combinados, filtros avanzados ni SQL; `group_by` continúa devolviendo resultados para todos los grupos.

El MVP sigue sin procesamiento paralelo y no debe asumirse que el repository en memoria es seguro ante accesos concurrentes.

## Errores y pruebas

- `400 Bad Request`: JSON inválido o pregunta vacía.
- `404 Not Found`: ID de dataset inexistente.
- `422 Unprocessable Entity`: pregunta no interpretable.
- `502 Bad Gateway`: Ollama inaccesible, timeout, respuesta inválida o instrucción rechazada.

Las pruebas usan `httptest` y un servidor mock; no requieren una instancia real de Ollama. Ejecutalas con:

```powershell
go test ./...
```

Las pruebas mock no verifican la calidad de interpretación del modelo real. Para comprobarla, ejecutá consultas con Ollama y `qwen3:4b` activos.

