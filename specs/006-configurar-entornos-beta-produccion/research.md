# Investigación y decisiones: entornos beta y producción

**Fecha**: 2026-10-17  
**Base**: `spec.md` aclarado y código existente. No se implementa código en esta fase.

## 1. Configuración y composición

- **Decisión**: un paquete `config/` carga el entorno una vez, produce `Config` tipada e inmutable para los consumidores y valida antes de abrir la base o construir proveedores. `app/` es la única raíz de composición. Los constructores reciben opciones concretas o puertos; ningún caso de uso, handler ni adaptador consulta `os.Getenv`.
- **Motivo**: hoy `app/app.go`, `adapters/output/datasource/connection.go`, middleware, casos de uso y adaptadores leen variables por separado. `USE_MOCK_PROVIDERS` no cubre WhatsApp ni Gemini y Mercado Pago puede caer silenciosamente a mock.
- **Alternativas**: Viper no aporta una necesidad comprobada; un lector pequeño con `os.LookupEnv` y `strconv` permite distinguir ausente de inválido. Un interruptor global de mocks no satisface FR-019/020. `godotenv` queda restringido a una carga local opcional, nunca implícita en el servicio desplegado; si `ENV_FILE` se declara explícitamente, un archivo ilegible o mal formado falla.

## 2. Límites de paquetes y orden de dependencias

- **Decisión**: conservar `core/domain` → `ports/{input,output}` → `core/usecase` como centro lógico; `adapters/input/http` y `adapters/output` implementan los puertos; `app` depende de todos para conectarlos; `main` solo arranca. Añadir `ports/output/agent_model.go`, `adapters/output/ai/` para Gemini real/simulado y adaptadores WhatsApp real/captura. El modelo de conversación que cruza al caso de uso deja de ser `genai.Content` y pasa a un tipo propio o del puerto; el SDK permanece en el adaptador.
- **Motivo**: `WhatsAppAgentUseCase` crea `genai.Client` y hace fallback de Gemini; los handlers dependen del cliente concreto de WhatsApp. La selección de modo debe ser visible y cerrada ante fallo.
- **Alternativas**: reorganizar todo el árbol o extraer GORM de cada entidad sería una reescritura excesiva. La importación de GORM en `core/domain/order.go` se registra como deuda existente; para esta función solo se cambiarán los bordes afectados por proveedores y configuración.

## 3. Persistencia y simuladores dinámicos

- **Decisión**: los simuladores beta mantienen operaciones de pago, cotizaciones, despachos, estados y mensajes salientes en la base beta, no en variables globales ni solo en RAM. Un puerto de almacenamiento específico permite probar los casos de uso; el adaptador puede usar GORM. IDs de prueba llevan prefijo inequívoco, se vinculan a pedido/dirección y se comprueban por importe, vencimiento y transición. La escritura de operaciones y cambios de estado se ofrece solo mediante rutas de operador autenticado, registradas únicamente en beta y solo para el proveedor simulado respectivo.
- **Motivo**: `MockPaymentProvider.VerifyPaymentStatus` aprueba cualquier ID; `SeedGlobalMockTransfer` mezcla instancias; `MockDeliveryProvider` ofrece una tarifa plana y carece de estado verificable. Los ensayos deben sobrevivir un reinicio beta y no depender de qué réplica atendió una solicitud.
- **Alternativas**: mocks en memoria sirven a pruebas unitarias, pero no al beta de Docker persistente. Reutilizar webhooks de proveedores para controles manuales confundiría firmas y autorización.

## 4. Conciliación, despacho y eventos

- **Decisión**: el control beta registra primero una operación ficticia `approved|pending|rejected` y, solo si está aprobada, solicita conciliación por `order_id` y `operation_id`; el caso de uso comprueba importe, moneda, estado e ID no consumido antes de marcar pagado. El despacho continúa por la ruta de operador existente. Un estado de envío propio guarda `pending → en_route|cancelled`, `en_route → delivered|cancelled`; el pedido pasa a `DELIVERED` o `CANCELLED` solo en los estados finales. Repeticiones idénticas son idempotentes; saltos o cambios tras estado final se rechazan.
- **Motivo**: la conciliación por monto sin pedido puede ser ambigua. `ApplyCourierStatus` actualmente omite `en_route` y tolera estados finales sin distinguir transiciones. La entrega requiere estado propio para no inventar un `OrderStatus` nuevo.
- **Alternativas**: aprobación automática y temporizadores fueron descartados por las aclaraciones del usuario.

## 4a. Webhooks, sesiones y escalado

- **Decisión**: al recibir un webhook externo, verificar firma y persistir un ID de evento/inbox antes de responder; procesar el trabajo largo mediante un disparador durable autenticado (Cloud Tasks en Cloud Run, ejecutor local beta). Historial de conversación necesario para respuestas posteriores y deduplicación de eventos viven en almacenamiento compartido. La sesión HTTP usa secreto inyectado; la cookie de producción exige `Secure` sobre HTTPS.
- **Motivo**: el handler WhatsApp actual responde 200 y continúa en una goroutine; Cloud Run puede suspender CPU o terminar la instancia tras la respuesta. Su memoria local para historial y deduplicación no se comparte entre réplicas. La conciliación existente también ignora errores de persistencia al marcar un pedido pagado, lo que debe corregirse junto con idempotencia transaccional.
- **Alternativas**: procesar Gemini antes de responder puede exceder el tiempo de confirmación del webhook; confiar solo en goroutines o timers locales no asegura entrega ni estado tras reinicio.

## 5. Identidades, proveedores externos de prueba y secretos

- **Decisión**: `APP_ENV=beta|production` es obligatorio. Modos independientes `mock|test` para WhatsApp, Gemini y Mercado Pago en beta, `mock` obligatorio para Uber y geocodificación en la primera entrega; producción exige modos `live` y credenciales completas. La validación se hace en dos etapas: forma y combinaciones locales antes de abrir recursos; comprobación de identidad/destino del recurso antes de aceptar tráfico o ejecutar migraciones. La DB contiene un marcador de entorno provisionado fuera de la migración y se comprueba junto con usuario/permisos; identidades de cuenta MP y número WA de beta se fijan mediante identificadores esperados y listas permitidas, con credenciales de test separadas. No se aceptan destinos arbitrarios solo porque una variable diga `test`; cuando un proveedor no permita verificar técnicamente su identidad desde la API, el despliegue exige identidad de servicio/proyecto y secreto beta separados más verificación operativa documentada.
- **Motivo**: el nombre de una base o el prefijo de un token no prueban aislamiento. Las credenciales beta no deben poder contactar cuentas operativas.
- **Alternativas**: seleccionar mock al faltar token y validar solo que `DB_NAME` contiene `beta` no cumplen la especificación. Un secreto real configurado en un proveedor simulado no se usa ni se imprime.

## 6. Base de datos y migraciones

- **Decisión**: `adapters/output/datasource` solo abre y comprueba la conexión recibida. `cmd/migrate` ejecuta un paso controlado e idempotente de esquema y datos iniciales, con registro de versión/resultado; el servicio jamás llama `AutoMigrate` ni reescribe claves administrativas al iniciar. Beta aplica ese paso a su contenedor exclusivo antes de levantar la API. Producción usa una ejecución separada con identidad de migración autorizada y promoción de revisión solo tras éxito.
- **Motivo**: hoy `Connect` combina `.env`, global `DB`, `AutoMigrate`, seeds y actualización de contraseña del admin. Cada réplica de Cloud Run no debe mutar la base durante su arranque.
- **Alternativas**: `AutoMigrate` en cada instancia es incompatible con FR-013; cambios manuales sin paso registrado son difíciles de verificar. El plan no supone que GORM `AutoMigrate` pueda revertir cambios destructivos; cada cambio de esquema requiere revisión de compatibilidad con la revisión anterior.

## 7. Contenedores, Cloud Run y archivos

- **Decisión**: una imagen Linux/amd64 con binario y recursos embebidos; Compose beta usa proyecto, red, MySQL, usuario, volumen y puerto de host exclusivos, sin enlaces a producción. `PORT` se valida y el servidor escucha `0.0.0.0:$PORT`. La conexión MySQL tipada admite exactamente TCP local o socket Unix de Cloud SQL; combina `DB_HOST/DB_PORT` o `DB_SOCKET`, nunca ambos. Secret Manager entrega secretos a la revisión, con acceso mínimo. Archivos de negocio persistentes usan un puerto de almacenamiento con adaptador local beta y Cloud Storage en producción, o se bloquea su función hasta que el almacenamiento externo esté configurado. La simulación pública `/chat` y `/webhook/whatsapp/simulate` se limita a beta/autorización; producción no la registra.
- **Motivo**: Cloud Run inyecta `PORT`, Cloud SQL ofrece socket Unix bajo `/cloudsql/...`, y el sistema de archivos de la instancia no garantiza persistencia. `app.Static` apunta hoy a `uploads` local. Sesiones y deduplicación de WhatsApp que deban sobrevivir réplicas también necesitan almacenamiento compartido o una política explícita de idempotencia.
- **Alternativas**: montar uploads locales como persistencia de producción y depender del proceso para estado beta no resisten reinicios/escalado.

## Fuentes y evidencia

- Proyecto: `app/app.go`, `adapters/output/datasource/connection.go`, `core/usecase/whatsapp_agent_usecase.go`, `core/usecase/payment_reconciliation_usecase.go`, `core/usecase/delivery_usecase.go`, `adapters/output/payment/mock_payment_provider.go`, `adapters/output/delivery/mock_delivery_provider.go`.
- Referencia del usuario: texto adjunto «La mejor práctica en Go…»; se adopta la configuración tipada y la inyección, pero se sustituye el interruptor global por modos independientes y no se aceptan defaults silenciosos de perfil/booleans.
- Google Cloud: [contrato de contenedor y PORT](https://docs.cloud.google.com/run/docs/troubleshooting), [conexión Cloud SQL MySQL](https://docs.cloud.google.com/sql/docs/mysql/connect-run), [Secret Manager en Cloud Run](https://docs.cloud.google.com/run/docs/configuring/services/secrets), [revisiones sin tráfico y promoción](https://docs.cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration), [persistencia de archivos](https://docs.cloud.google.com/run/docs/fit-for-run), [Cloud Run Jobs](https://cloud.google.com/run/docs/create-jobs).
