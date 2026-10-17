# Contratos HTTP de control beta

Estos endpoints se registran **solo** si `APP_ENV=beta` y el proveedor correspondiente está en `mock`. Requieren la cookie de sesión de operador `ADMIN` o `EMPLOYEE`, con secreto beta inyectado. No reutilizan webhooks de Meta, Mercado Pago o Uber ni aceptan sus firmas como autorización. Todas las respuestas llevan `Cache-Control: no-store`. Los IDs y URLs de prueba son inequívocos.

## Registrar operación ficticia

`POST /api/beta/payments/operations`

```json
{
  "order_id": 42,
  "operation_id": "beta-pay-example-42",
  "amount": "8500.00",
  "currency": "ARS",
  "status": "approved",
  "payer_name": "Comprador Prueba",
  "payer_id": "test-42"
}
```

`operation_id` puede omitirse y se genera. `amount` es decimal textual de dos cifras para evitar ambigüedad de coma flotante en esta interfaz. `status` es obligatorio: `approved|pending|rejected`. La operación queda asociada a `order_id`; si está aprobada, se invoca conciliación para **ese pedido y esa operación**, nunca se marca pagado directamente ni se busca otro pedido por importe. Pendiente/rechazada se registra pero no se concilia como aprobada.

**201**:

```json
{
  "operation_id": "beta-pay-example-42",
  "order_id": 42,
  "status": "approved",
  "simulated": true,
  "reconciliation": {"status": "MATCHED", "order_id": 42},
  "order_status": "PAID"
}
```

Una aprobada con monto incorrecto responde **201** con `reconciliation.status=AMOUNT_MISMATCH` y pedido pendiente; registrar el hecho ficticio no implica acreditarlo. Pendiente/rechazada responde **201** con `reconciliation: null` y pedido sin pagar. Repetir el mismo `operation_id` con exactamente el mismo cuerpo es idempotente (**200** con resultado actual); usarlo con contenido distinto da **409**. Pedido inexistente o que no espera pago: **404/409**; cuerpo inválido: **400**; sesión ausente: **401**; rol inválido: **403**; proveedor no simulado o perfil producción: **404**. La respuesta jamás dice que se recibió dinero real.

## Consultar una operación

`GET /api/beta/payments/operations/:operation_id` → **200** con `operation_id`, `order_id`, `amount`, `currency`, `status`, `simulated: true`, `reconciliation_status`; **404** si no existe. Sirve para ensayar consulta y verificar persistencia tras reiniciar beta.

## Cambiar estado de envío ficticio

`POST /api/beta/deliveries/:external_delivery_id/transitions`

```json
{"status": "en_route"}
```

`pending` se crea automáticamente al despachar manualmente por la ruta existente `POST /api/orders/:id/dispatch-delivery`; nunca se crea mediante este endpoint. El contrato admite solo `en_route|delivered|cancelled`, conforme a la tabla de [data-model.md](../data-model.md). **200**:

```json
{
  "order_id": 42,
  "external_delivery_id": "beta-del-42",
  "delivery_status": "en_route",
  "order_status": "DISPATCHED",
  "tracking_url": "http://localhost:8081/api/beta/deliveries/beta-del-42",
  "simulated": true
}
```

Transición inválida, salto de estado o cambio posterior a estado final: **409**. ID desconocido: **404**. Estado desconocido: **400**. Sesión/rol inválido: **401/403**. Perfil o modo incompatible: **404**. Una repetición exacta de la transición ya aplicada puede devolver **200** con estado actual; la implementación debe registrar una clave de idempotencia o versión para no aceptar una transición distinta por carrera.

`GET /api/beta/deliveries/:external_delivery_id` → **200** con los campos anteriores, cotización usada, fechas y estado actual; **404** si no existe. El tracking URL apunta al beta local, nunca a una URL de Uber.

## Conversación y salidas capturadas

En beta, `POST /chat` o `POST /webhook/whatsapp/simulate` conserva la forma actual `{phone,name,message,reset}`, pero requiere la misma autorización de operador y producción no registra ninguna de estas rutas. Mensajes salientes simulados se consultan mediante `GET /api/beta/messages?recipient=...` para comprobar contenido sin contactar Meta. Esta consulta requiere operador y devuelve solo datos de prueba con retención y redacción. Las respuestas sobre pagos identifican explícitamente la simulación.

## Webhooks y errores comunes

Los webhooks existentes `/webhook/whatsapp`, `/webhook/mercadopago`, `/webhook/uber-delivery` siguen separados de los controles beta. En modo externo `test|live` se verifican los secretos/firmas del perfil antes de conciliar o aplicar un estado; secreto vacío es un error de arranque. En modo `mock`, un webhook externo no acredita operaciones ni cambia envíos. No hay fallback que apruebe un ID de pago desconocido. Los errores no incluyen tokens, cuerpos firmados ni contraseñas.
