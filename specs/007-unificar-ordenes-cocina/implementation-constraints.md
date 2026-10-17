# Restricciones técnicas entregadas para la planificación

Este documento conserva las decisiones técnicas explícitas del pedido. La especificación funcional describe el comportamiento observable; el plan deberá implementar estos contratos.

## Arquitectura

- Mantener las capas `core/domain`, `core/usecase`, `ports/input`, `ports/output`, `adapters/input` y `adapters/output` y la inversión de dependencias.
- Definir `domain.OrderEvent` con tipos `ORDER_CREATED`, `ORDER_PAID`, `ORDER_STATUS_UPDATED`, `ORDER_DISPATCHED` y `ORDER_CANCELLED`; incluir la orden hidratada y marca de tiempo ISO-8601 en zona Córdoba.
- Añadir `ports/output.OrderEventPublisher` con `Publish(event domain.OrderEvent)`.
- Ampliar el puerto de órdenes con `ConfirmOrderPayment(ctx, orderID, externalPaymentID)`, `UpdateOrderStatus(ctx, orderID, newStatus)` y `DispatchOrderDelivery(ctx, orderID)`.
- Emitir eventos después de persistir la transición. La hidratación debe incluir `Customer`, `Items.Topping` y `Address`.

## Distribución de novedades

- Implementar `adapters/input/http/sse/order_sse_broker.go` con suscriptores `map[chan domain.OrderEvent]struct{}`, `sync.RWMutex` y capacidad de 16 eventos por suscriptor.
- Publicar con envío no bloqueante; un canal lleno se omite sin frenar ventas ni goroutines.
- `GET /api/orders/stream` requiere el middleware `Protected()` y la cookie JWT `session_token`.
- Usar `text/event-stream`, `no-cache`, `keep-alive`, `X-Accel-Buffering: no`; enviar `: keepalive\n\n` y vaciar cada 25 segundos.
- Mantener `GET /api/orders?status=PENDING_PAYMENT,PAID,PREPARING,DISPATCHED` como fuente de resincronización.
- Un broker en memoria no comparte eventos entre réplicas de Cloud Run. Configurar una sola instancia por entorno para cumplir la difusión a todas las pantallas; documentar timeout y reconexión de `EventSource`. La consulta completa recupera eventos perdidos.

## Contratos HTTP

- `POST /api/orders`: default `PAID`; permitir explícitamente `PENDING_PAYMENT`, `PAID` y `COMPLETED`.
- `POST /api/orders/:id/confirm-payment`: aprobación manual autorizada.
- `PATCH /api/orders/:id/status`: `PREPARING`, `READY_FOR_PICKUP`, `COMPLETED`, `CANCELLED`, según transición.
- `POST /api/orders/:id/dispatch`: despacho bajo demanda.
- Registrar los tres endpoints de mutación y el stream en `url_mappings.go` con `middleware.Protected()`.

## Integridad y fallas

- Confirmación de pago, incremento de `TotalOrders` / `FirstOrderAt` y efecto de caja deben ser atómicos e idempotentes. El estado previo se verifica dentro del límite transaccional.
- Crear `PENDING_PAYMENT` persiste orden e ítems sin reconocer ingreso ni compra; `PAID` y `COMPLETED` sí los reconocen.
- El despacho exige `DELIVERY` y estado `PAID` o `PREPARING`; renovar cotización de Uber Direct cuando no exista o supere 15 minutos.
- Prevenir dos llamadas `CreateDelivery` concurrentes mediante reserva persistente por orden o mecanismo equivalente. Una verificación de `Status != DISPATCHED` previa a llamar al proveedor es insuficiente por sí sola.
- La integración con Uber debe permitir reconciliar una llamada externa cuyo resultado local sea incierto, antes de reintentar.
- La notificación al cliente usa el mensaje “🚚✨ ¡Canerollss en camino! ✨” con el `TrackingURL`; el fallo de WhatsApp no revierte un despacho ya aceptado.
- En beta deben usarse las dependencias y credenciales beta; en producción, las productivas. El entorno no debe cambiar las reglas comerciales.

## Decisiones para revisar en planificación

- Verificar si los proveedores actuales permiten idempotencia/reconciliación de creación de delivery.
- Revisar cómo calcula caja los ingresos: vínculo de orden a caja y cobro reconocido pueden ser conceptos distintos.
- Determinar el permiso existente para aprobar pagos y operar cocina, sin ampliar autorización por accidente.
