# Guía de validación prevista

Esta guía describe cómo **se validará** la función una vez implementada. En esta fase aún no existen `Dockerfile`, `compose.beta.yml`, el comando de migración ni las rutas `/api/beta/*`; por eso estos comandos son el contrato de ejecución esperado, no una afirmación de que hoy funcionen.

## Prerrequisitos

- Docker con Compose y Go 1.26 para pruebas de código.
- Puertos libres para beta: se propone `8081` en el host y MySQL beta solo dentro de su red Compose. Inventariar antes los puertos, contenedores, bases y volúmenes de la operación real; no detenerlos ni reutilizarlos.
- Copiar `.env.beta.example` a un archivo local ignorado `.env.beta`, con `APP_ENV=beta`, cinco proveedores `mock`, contraseñas ficticias únicas y `DB_EXPECTED_ID` de la instancia beta. No copiar credenciales reales.

## Levantar beta

```sh
docker compose -p canerollss-beta -f compose.beta.yml up -d db
docker compose -p canerollss-beta -f compose.beta.yml run --rm migrate
docker compose -p canerollss-beta -f compose.beta.yml up -d api
docker compose -p canerollss-beta -f compose.beta.yml ps
```

Resultado esperado: API en `http://localhost:8081`; DB y volumen identificados con `canerollss-beta`, sin puerto de MySQL publicado al host ni recursos compartidos con producción. Los logs muestran `profile=beta` y modos `mock`, sin DSN ni tokens. El comando `migrate` termina antes de iniciar la API y puede repetirse sin resetear datos.

## Probar pedido, pago y reparto

1. Autenticarse como operador beta por `POST /api/auth/login` con `{ "username": "...", "password": "..." }` y guardar la cookie `session_token`. Confirmar que la cookie beta no sirve para el servicio real.
2. Usar los endpoints existentes para abrir caja si hace falta, crear cliente/dirección validada y crear pedido de entrega. Anotar `order_id`, importe exacto y estado `PENDING_PAYMENT`. La forma de estos recursos sigue en los DTO actuales; la nueva interfaz de operación ficticia está en [beta-http.md](contracts/beta-http.md).
3. Registrar una operación `pending` por `POST /api/beta/payments/operations`; el pedido sigue sin pagar. Repetir con una operación `rejected` y luego con una `approved` de monto diferente: ninguna paga el pedido.
4. Registrar una operación `approved` por el importe exacto con otro `operation_id`. Comprobar `MATCHED` y pedido `PAID`. Reiniciar solo `api`; consultar la operación y comprobar que sigue disponible. Confirmar que no se llamó a Mercado Pago.
5. Usar `POST /api/orders/:id/dispatch-delivery` con cookie de operador. Debe producir ID/cotización/tracking `beta-`, sin llamar a Uber. Comprobar que un pedido sin pagar o sin dirección válida no se despacha.
6. Cambiar el envío `pending → en_route → delivered` mediante `POST /api/beta/deliveries/:external_delivery_id/transitions`. Verificar pedido `DISPATCHED` durante `en_route` y `DELIVERED` al final; un cambio posterior devuelve 409. Repetir otro pedido con cancelación. Reiniciar la API y consultar el estado.
7. Enviar una conversación por la ruta de simulación autorizada, inspeccionar `GET /api/beta/messages` y comprobar respuesta reproducible y marcada como prueba. No deben salir mensajes a números reales.

## Comprobar aislamiento y errores de arranque

- Ejecutar `go test ./...` y las pruebas de integración beta. Las pruebas de red real requieren activación explícita.
- Cambiar por separado `WHATSAPP_MODE`, `GEMINI_MODE` y `MP_MODE` a `test` con credenciales/identidades beta verificadas; probar las ocho combinaciones y comprobar que solo el proveedor pedido hace llamadas externas. `UBER_MODE` permanece `mock`.
- Retirar una credencial obligatoria, poner `PORT=abc`, `APP_ENV` desconocido, dos transportes DB a la vez o identidad DB productiva en beta. Cada caso debe fallar antes de aceptar HTTP, nombrando solo el campo.
- Intentar usar rutas `/api/beta/*` sin cookie y con rol ajeno: 401/403. Con perfil producción: 404. Las rutas públicas de simulación tampoco deben existir en producción.
- Reiniciar y detener beta diez veces; comprobar que la operación real sigue disponible y sus datos no cambiaron. Detener con `docker compose -p canerollss-beta -f compose.beta.yml down`; no usar `-v` si se desea conservar datos beta.

## Ensayo de despliegue en Google Cloud

1. Construir una sola imagen por versión y probar su ejecución beta. Configurar una revisión de ensayo con `APP_ENV=production`, Secret Manager y una base de **preproducción** aislada; no usar datos reales para el ensayo.
2. Ejecutar el comando de migración como Cloud Run Job o paso separado con identidad de esquema. Forzar un fallo controlado y confirmar que la nueva revisión no recibe tráfico. Repetir con éxito y verificar la versión de esquema.
3. Desplegar la misma imagen en revisión sin tráfico, verificar `PORT`, socket Cloud SQL, permisos, salud, webhooks firmados, almacenamiento persistente y ausencia de secretos en logs. Promocionar tráfico solo tras éxito. Reiniciar varias instancias y confirmar que ninguna migra ni reescribe credenciales.

Referencias operativas: [configuration.md](contracts/configuration.md), [data-model.md](data-model.md), [research.md](research.md). El despliegue real y la migración de datos de producción no están autorizados por esta especificación.
