Actúa como un desarrollador senior especializado en Go e integración de modelos de lenguaje.

Quiero desarrollar el proyecto CSV Copilot. El documento [CONTEXT.md](http://CONTEXT.md) contiene los requisitos funcionales, la arquitectura, la estructura de carpetas, las operaciones permitidas y los criterios de aceptación. Léelo y respétalo como especificación principal.

Trabajaremos de forma incremental. En esta primera etapa:

1. Inspecciona el repositorio actual y verifica si ya existe código.
2. Crea únicamente el esqueleto inicial del proyecto, respetando la estructura definida en [CONTEXT.md](http://CONTEXT.md).
3. Inicializa el módulo Go y crea el punto de entrada en cmd/app/main.go.
4. Implementa la carga y validación básica de configuración.
5. Implementa un servidor HTTP mínimo con un endpoint GET /health que devuelva una respuesta JSON.
6. Añade pruebas para las funcionalidades implementadas.
7. Comprueba que el proyecto compila y que las pruebas pasan.

Todavía no integres el modelo de lenguaje, no implementes todas las operaciones de análisis y no construyas la interfaz web.

Antes de modificar archivos, explícame brevemente el plan. Después realiza los cambios, ejecuta las pruebas disponibles y resume qué archivos creaste, qué decisiones tomaste y cómo ejecutar el proyecto.

No inventes resultados de pruebas. Si no puedes ejecutar algún comando, indícalo explícitamente.

Prioriza simplicidad, código idiomático de Go, separación de responsabilidades y dependencias mínimas. No modifiques ni elimines código existente sin una razón justificada.