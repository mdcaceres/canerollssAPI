# Modelo de datos y estados

Este diseño amplía los modelos existentes; no contiene migraciones ni implementación. Las tablas de prueba residen exclusivamente con datos en la DB beta. La misma versión de esquema puede existir en producción, pero los controles beta no se registran allí.

## Entidades de configuración (en memoria)

### Configuración validada

| Campo | Tipo | Regla |
| --- | --- | --- |
| `profile` | enum `beta|production` | obligatorio; no hay perfil implícito |
| `listenPort` | entero 1–65535 | `PORT` inyectado o valor local documentado; valor inválido falla |
| `database` | `DatabaseConfig` | un solo transporte TCP o socket; identidad del recurso comprobada |
| `providers` | cinco modos tipados | combinaciones de [configuration.md](contracts/configuration.md) |
| `auth`, `store`, `delivery`, `files` | structs tipados | secretos y reglas validados por perfil |

Se produce una vez al arrancar. No se guarda en DB ni se imprime completa. Los componentes reciben solo la sección necesaria por constructor.

### Identidad de recurso

`profile`, `resourceKind` (DB, número WhatsApp, cuenta MP, proyecto/clave Gemini, Maps, Uber, archivos), `expectedID`, `observedID`, `verificationStatus`. Es una comprobación de arranque y del procedimiento de despliegue, no una tabla de secretos. Si una identidad requerida no puede verificarse o discrepa, no se acepta tráfico.

## Entidades persistentes existentes

### Pedido (`core/domain.Order`)

Campos relevantes: `ID`, `Status`, `TotalAmount`, `PaymentMethod`, `ExternalPaymentID`, `AddressID`, `DeliveryQuoteID`, `DeliveryQuoteExpiresAt`, `ExternalDeliveryID`, `TrackingURL`, `DeliveryProviderFee`. Estados existentes `PENDING_PAYMENT`, `PAID`, `PREPARING`, `DISPATCHED`, `DELIVERED`, `COMPLETED`, `CANCELLED`.

Reglas conservadas: una operación aprobada, no consumida, en ARS y por importe exacto puede pagar el pedido mediante conciliación; el despacho requiere operador, pago y dirección validada. El estado `en_route` pertenece al envío simulado, mientras el pedido conserva `DISPATCHED`.

## Entidades beta persistentes

### Operación de pago ficticia

| Campo | Tipo | Regla |
| --- | --- | --- |
| `operationID` | string único, prefijo `beta-pay-` | generado si no se envía; idempotencia por ID |
| `orderID` | referencia a pedido | pedido beta existente y pendiente |
| `amount` | importe decimal en unidades menores o decimal exacto | positivo; comparación exacta con total del pedido, sin redondeo float nuevo |
| `currency` | enum `ARS` | obligatorio |
| `status` | enum `approved|pending|rejected` | explícito; sin valor predeterminado |
| `payerName`, `payerID` | string opcional | datos ficticios |
| `createdAt`, `approvedAt`, `createdBy` | marcas/auditoría | `approvedAt` solo para aprobada; operador autenticado |
| `reconciledOrderID` | referencia opcional | solo si conciliación dio `MATCHED` |

Una operación pertenece a un solo pedido; no puede pagar otro ni cambiar de importe/estado una vez registrada. Repetir el mismo ID y contenido devuelve el mismo resultado; contenido distinto produce conflicto. `pending` o `rejected` nunca se ofrecen como acreditadas. La búsqueda de transferencias aprobadas del puerto existente lee esta entidad, limitada a beta.

### Preferencia de pago ficticia

`preferenceID` con prefijo `beta-pref-`, `orderID`, `createdAt`, `testURL`. La URL identifica claramente una pantalla/ruta de prueba del mismo entorno, sin apuntar al checkout real. Crear preferencia no crea operación aprobada.

### Cotización de reparto ficticia

`quoteID` con prefijo `beta-quote-`, `orderID` cuando la cotización se asigna al pedido, `addressID` y huella de dirección normalizada/geocodificada, `distanceKm`, `feeARS`, `durationMinutes`, `createdAt`, `expiresAt`, `consumedAt`. Distancia calculada desde tienda validada; importe y duración varían mediante fórmula determinista documentada durante implementación. La cotización no puede usarse vencida, con otra dirección o por otro pedido; consumo único.

### Envío ficticio

`externalDeliveryID` con prefijo `beta-del-`, `orderID` único, `quoteID`, `addressID`, `trackingURL` local beta, `status` enum `pending|en_route|delivered|cancelled`, `createdAt`, `updatedAt`, `version`. Se crea solo tras la ruta de despacho del operador y una cotización vigente. `version` o bloqueo transaccional protege de dos transiciones concurrentes.

| Estado anterior | Acción del operador | Estado siguiente | Estado del pedido |
| --- | --- | --- | --- |
| sin envío | despacho válido | `pending` | `DISPATCHED` |
| `pending` | `en_route` | `en_route` | `DISPATCHED` |
| `pending` | `cancelled` | `cancelled` | `CANCELLED` |
| `en_route` | `delivered` | `delivered` | `DELIVERED` |
| `en_route` | `cancelled` | `cancelled` | `CANCELLED` |

Repetir el mismo estado con la misma clave idempotente devuelve el estado actual; un salto, cambio de estado final o ID ajeno devuelve conflicto. Ninguna transición se programa por tiempo.

### Mensaje saliente capturado

`messageID`, `recipient` ficticio, `body` redactado/limitado, `origin`, `createdAt`, `deliveryMode=mock`. Solo sirve para inspección de prueba; nunca se envía a Meta. Acceso limitado a operador beta y política de retención documentada. Los mensajes al cliente en beta deben señalar que son simulados cuando hablen de dinero o reparto.

### Evento entrante y sesión conversacional

El evento entrante conserva `provider`, `externalEventID` único, `profile`, `receivedAt`, `processingStatus`, `attempts` y referencia a payload protegido; se registra después de verificar firma y antes de confirmar el webhook. El procesamiento es idempotente y la cola puede reintentarlo. La sesión conversacional guarda `recipient`/identidad de prueba o cliente, historial en tipos propios (sin `genai.Content`), `updatedAt` y versión para concurrencia. Su almacenamiento es compartido por las instancias de producción; la política de retención y minimización de datos se documenta antes del despliegue.

## Datos iniciales y ciclo de vida

El comando de migración mantiene una versión/registro de ejecución, crea tablas necesarias y aplica datos iniciales idempotentes. En beta, el volumen MySQL conserva pedidos y entidades simuladas al reiniciar la API; destruir el volumen es una acción explícita documentada. En producción, usuario administrador inicial requiere secreto suministrado al paso controlado y nunca se reescribe al iniciar una instancia. Archivos persistentes no se guardan en el filesystem de Cloud Run.
