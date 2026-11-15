# Guía de clonación y ejecución local

Esta guía describe cómo obtener CSV Copilot desde Git y ejecutar el backend, Ollama y la interfaz web en una computadora local.

## Requisitos

- Git.
- Go 1.26 o superior.
- Ollama.
- El modelo `qwen3:4b` descargado en Ollama.

Comprobá las instalaciones:

```powershell
git --version
go version
ollama --version
```

## Clonar el proyecto

En PowerShell, ubicáte en la carpeta donde quieras guardar el proyecto y ejecutá:

```powershell
git clone https://github.com/MfKrp/Interprete-CSV-GOLANG.git
cd Interprete-CSV-GOLANG
```

Descargá las dependencias de Go y el modelo:

```powershell
go mod download
ollama pull qwen3:4b
```

La configuración predeterminada usa el puerto `8080`, Ollama en `http://localhost:11434`, el modelo `qwen3:4b` y un máximo de carga de 10 MB. El archivo `.env.example` muestra las variables disponibles, pero Go lee las variables del entorno del proceso; no carga automáticamente un archivo `.env`.

Para modificar una configuración solo para la sesión actual de PowerShell, por ejemplo:

```powershell
$env:PORT = "8080"
$env:OLLAMA_URL = "http://localhost:11434"
$env:OLLAMA_MODEL = "qwen3:4b"
$env:OLLAMA_TIMEOUT_SECONDS = "60"
$env:MAX_UPLOAD_SIZE_MB = "10"
```

## Iniciar la aplicación

Dejá Ollama ejecutándose. Si Ollama no se inició automáticamente como servicio, abrí una terminal y ejecutá:

```powershell
ollama serve
```

En otra terminal, entrá a la carpeta clonada e iniciá el backend:

```powershell
cd Interprete-CSV-GOLANG
go run ./cmd/app
```

Cuando aparezca el mensaje indicando que CSV Copilot escucha en `:8080`, abrí [http://localhost:8080/](http://localhost:8080/) en el navegador. Esa dirección sirve la interfaz web y la API.

Comprobá el estado del servidor desde otra terminal:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

La respuesta esperada indica que el servicio está activo, por ejemplo `status: ok`.

## Probar una carga y consulta desde PowerShell

Desde la carpeta del repositorio, cargá el CSV de demostración. El ID se genera en cada carga; guardá el que devuelve el servidor:

```powershell
$upload = curl.exe -sS -F "file=@ventas_demo.csv" http://localhost:8080/api/datasets | ConvertFrom-Json
$upload | ConvertTo-Json -Depth 5
$id = $upload.id
```

Recuperá los metadatos con ese ID:

```powershell
Invoke-RestMethod "http://localhost:8080/api/datasets/$id" | ConvertTo-Json -Depth 5
```

Hacé una consulta en lenguaje natural. El siguiente ejemplo cuenta vendedores distintos:

```powershell
$body = @{ question = "¿Cuántos vendedores hay?" } | ConvertTo-Json -Compress
$result = Invoke-RestMethod -Method Post `
  -Uri "http://localhost:8080/api/datasets/$id/query" `
  -ContentType "application/json; charset=utf-8" `
  -Body ([Text.Encoding]::UTF8.GetBytes($body))
$result | ConvertTo-Json -Depth 8
```

Con `ventas_demo.csv`, la cantidad de vendedores distintos es 4. También podés consultar, por ejemplo, “¿Cuántas ventas superan los 10000?” o “¿Cuánto vendió cada vendedor?”.

La carga guarda los datasets temporalmente en memoria. Si detenés o reiniciás el backend, los IDs anteriores dejan de existir: cargá de nuevo el CSV y usá el ID nuevo.

## Ejecutar pruebas y compilar

Desde la raíz del repositorio:

```powershell
go test ./...
go build ./...
```

Las pruebas automatizadas simulan Ollama cuando corresponde y no requieren tener Ollama activo. La demostración manual sí necesita Ollama iniciado y `qwen3:4b` instalado.

## Linux y macOS

Los pasos son los mismos. Usá un shell en lugar de PowerShell para clonar e iniciar:

```sh
git clone https://github.com/MfKrp/Interprete-CSV-GOLANG.git
cd Interprete-CSV-GOLANG
go mod download
ollama pull qwen3:4b
go run ./cmd/app
```

En otra terminal, iniciá `ollama serve` si Ollama no está corriendo como servicio. La interfaz queda en `http://localhost:8080/`. Para cambiar variables de entorno en esa terminal, por ejemplo `PORT`, usá `export PORT=8080` antes de iniciar Go.

## Problemas frecuentes

- **El puerto 8080 está ocupado:** cerrá el proceso que lo usa o cambiá `PORT` antes de arrancar el backend.
- **Ollama no está disponible:** comprobá que `ollama serve` esté ejecutándose y que `OLLAMA_URL` apunte a la instancia local.
- **No se encuentra el modelo:** ejecutá `ollama pull qwen3:4b` y verificá con `ollama list`.
- **`dataset not found`:** volvé a cargar el CSV y usá exactamente el `id` recibido en la respuesta de carga; los datasets solo viven mientras ese proceso del backend sigue activo.
- **No se abre la interfaz:** confirmá que el backend arrancó sin errores y que accedés a `http://localhost:8080/`.
