# Tasks: Entornos beta y producción aislados

**Input**: `specs/006-configurar-entornos-beta-produccion/{spec.md,plan.md,research.md,data-model.md,contracts/,quickstart.md}`.  
**Estado**: planificación de tareas; ningún elemento de esta lista se ha implementado por generar este archivo.

## Formato

Cada entrada tiene `- [ ] T### [P?] [US?] acción en ruta`. `[P]` indica archivos distintos y ausencia de una dependencia pendiente; las etiquetas `[US1]` a `[US5]` corresponden a las historias del spec. Los criterios y reglas completos están en los contratos enlazados arriba.

## Phase 1: Setup (infraestructura compartida)

**Propósito**: preparar un artefacto único y proteger archivos locales de credenciales.

- [ ] T001 Crear `Dockerfile` multietapa para Go 1.26 que produzca la API y la entrada de migración desde la misma revisión Linux/amd64, sin incorporar `.env` ni secretos.
- [ ] T002 [P] Actualizar `.dockerignore` y `.gitignore` para excluir `.env.beta`, secretos, DB/volúmenes locales, artefactos y archivos de prueba, conservando `.env.beta.example`.
- [ ] T003 [P] Registrar en `docs/environment-boundaries.md` los recursos actuales que deben inventariarse antes de usar beta (puertos, contenedores, DB y volúmenes), como plantilla sin copiar valores confidenciales.

---

## Phase 2: Foundational (bloquea todas las historias)

**Propósito**: carga tipada, identidad de datos, migración separada y composición con errores. Ninguna historia debe empezar hasta terminar esta base.

- [ ] T004 Crear `config/config.go` con structs tipados para perfil, puerto, DB, autenticación, tienda, archivos y cinco modos de proveedor; representar solo `beta|production` y modos de `contracts/configuration.md`.
- [ ] T005 Implementar `config/load.go` como único lector de variables, con precedencia del proceso sobre archivo local opcional; `ENV_FILE` declarado y mal formado/ilegible falla, mientras un archivo local no declarado y ausente no es obligatorio.
- [ ] T006 Implementar `config/validate.go` para rechazar perfil ausente/desconocido, booleanos/números/puerto 1–65535 inválidos, credenciales requeridas faltantes, transporte DB TCP/Unix simultáneo, modos prohibidos y flags legados ambiguos; exigir `ADMIN_PASSWORD` solo al proceso de migración inicial, no a la API; errores solo con nombre de campo.
- [ ] T007 [P] Crear `config/config_test.go` con casos de tabla para ausencia frente a valor inválido, precedencia de archivo, perfil obligatorio, booleans estrictos, secretos ocultos y combinaciones del contrato.
- [ ] T008 Refactorizar `adapters/output/datasource/connection.go` para recibir configuración DB tipada, abrir/ping MySQL o SQLite de prueba y devolver `(*gorm.DB,error)` sin global `DB`, `godotenv`, `AutoMigrate` ni seeds.
- [ ] T009 Implementar `adapters/output/datasource/identity.go` para cotejar `DB_EXPECTED_ID` con marcador provisionado y usuario/alcance observados antes de cualquier escritura; marcador ausente o cruzado devuelve error.
- [ ] T010 Crear `cmd/migrate/main.go` y `adapters/output/datasource/migrations.go` como paso versionado, verificable e idempotente de esquema/datos iniciales, incluidas operaciones/preferencias beta, cotizaciones/envíos, mensajes capturados, inbox y sesiones de `data-model.md`; separado del HTTP y sin rotar contraseña administrativa durante arranque.
- [ ] T011 [P] Crear `adapters/output/datasource/migrations_test.go` para repetición idempotente, marcador incorrecto, migración fallida con exit code no cero y ausencia de migraciones al construir la API.
- [ ] T012 Inyectar el secreto JWT tipado en `core/usecase/auth_usecase.go`, `adapters/input/http/middleware/auth_middleware.go` y `adapters/input/http/handlers/auth_handler.go`; conservar autorización `ADMIN|EMPLOYEE`, adaptar `tests/e2e_test.go` a los constructores y usar cookie `Secure` en producción.
- [ ] T013 Refactorizar `app/app.go` y `main.go` para que `config.Load/Validate` y la verificación DB precedan a repositorios/HTTP, los constructores devuelvan errores y Fiber escuche en `0.0.0.0:$PORT`; no iniciar parcialmente.

**Checkpoint**: una configuración inválida o DB equivocada falla antes de aceptar tráfico y la API nunca migra al arrancar.

---

## Phase 3: User Story 1 — Probar beta sin afectar ventas (P1, MVP)

**Goal**: iniciar beta local con DB/volumen/red/puerto propios y cero llamadas a cuentas operativas.

**Independent Test**: iniciar beta junto al servicio real, crear un pedido de prueba, reiniciar/detener beta y constatar datos reales intactos y datos beta persistentes.

### Verificación primero

- [ ] T014 [P] [US1] Escribir `tests/beta_isolation_test.go` para comprobar DB exclusiva, marker beta, persistencia tras reinicio y que las peticiones beta no llegan a endpoints externos de pago, Maps, WhatsApp, Gemini o Uber.
- [ ] T015 [P] [US1] Escribir `tests/beta_coexistence_test.go` para comprobar puerto beta diferente, ciclo de vida independiente y ausencia de cambios en una base real de ensayo; no conectar a la DB operativa.

### Implementación

- [ ] T016 [P] [US1] Crear `compose.beta.yml` con servicios `db`, `migrate`, `api`, proyecto/volumen/red beta y puerto host 8081, sin publicar MySQL ni montar recursos de producción; `api` no inicia antes de una migración exitosa.
- [ ] T017 [P] [US1] Crear `deploy/beta/mysql-init.sql` para usuario restringido, DB exclusiva y marcador de identidad beta inicializados antes de migrar; no incluir contraseñas reales.
- [ ] T018 [P] [US1] Crear `.env.beta.example` con `APP_ENV=beta`, los cinco modos `mock`, IDs/contraseñas ficticias, puerto interno 8080 y valores de tienda/radio válidos según `contracts/configuration.md`.
- [ ] T019 [P] [US1] Definir `ports/output/agent_model.go` con mensajes/historial propios, sin importar `google.golang.org/genai`, para que el agente pueda recibir Gemini o simulador.
- [ ] T020 [US1] Implementar `adapters/output/ai/mock_agent.go` con respuestas reproducibles para catálogo, pedido y consulta de pago; nunca afirmar acreditación real ni realizar llamadas salientes.
- [ ] T021 [P] [US1] Crear `ports/output/message_store.go`, `adapters/output/repository/mock_message_repository.go` y `adapters/output/whatsapp/mock_sender.go` para capturar mensajes beta sin enviarlos, con `messageID`, destinatario ficticio, cuerpo limitado, origen, fecha y `deliveryMode=mock`.
- [ ] T022 [US1] Sustituir tipos Gemini en `core/usecase/whatsapp_agent_usecase.go` y `adapters/input/http/session/session_manager.go` por el puerto/modelo propios, adaptando `adapters/input/http/session/session_manager_test.go` y manteniendo historial multi-turno reproducible.
- [ ] T023 [US1] Corregir `adapters/output/payment/mock_payment_provider.go` para que un ID no registrado nunca sea `approved`, eliminar el ledger global y adaptar los fixtures de `tests/reconciliation_webhook_test.go` a un proveedor inyectado; no hay acreditación al crear pedido.
- [ ] T024 [US1] Conectar en `app/app.go` los cinco adaptadores mock de beta por defecto, el sender capturado y la DB beta inyectada; impedir construcción de cliente real en esta ruta de arranque.
- [ ] T025 [US1] Ajustar `adapters/input/http/routes/webhook_routes.go` para que `/chat` y `/webhook/whatsapp/simulate` requieran operador en beta y no se registren en producción.
- [ ] T026 [US1] Completar `tests/beta_isolation_test.go` con pedido vía API y reinicio de `api` sin borrar el volumen; verificar que el servicio de ensayo real continúa disponible.
- [ ] T027 [US1] Documentar los comandos mínimos de arranque, migración, inspección y parada segura en `docs/beta-local.md` y enlazarlos desde `specs/006-configurar-entornos-beta-produccion/quickstart.md`.

**Checkpoint**: beta puede ejecutar un pedido básico aislado desde un checkout limpio sin credenciales externas.

---

## Phase 4: User Story 2 — Arrancar solo con configuración segura (P1)

**Goal**: elegir cada proveedor una vez y fallar antes de tráfico ante modo, credencial o destino inválido.

**Independent Test**: arrancar perfiles válidos, retirar o malformar cada campo crítico y probar las ocho combinaciones independientes de WhatsApp/Gemini/MP test.

### Verificación primero

- [ ] T028 [P] [US2] Escribir `config/provider_modes_test.go` para la matriz completa: beta `mock|test` individual en WhatsApp/Gemini/MP, Uber y Maps `mock`; producción cinco `live`; rechazos de modo imposible y flags legados.
- [ ] T029 [P] [US2] Escribir `app/startup_test.go` para credencial/destino incompleto, identidad DB cruzada, archivo local inválido, proveedor real sin secreto y logs sin valores sensibles antes de escuchar.

### Implementación

- [ ] T030 [P] [US2] Cambiar constructores de `adapters/output/delivery/uber_direct_provider.go`, `adapters/output/delivery/mock_delivery_provider.go` y `adapters/output/geocoding/google_maps_provider.go` para recibir tienda, cobertura, credenciales y destinos tipados; eliminar lecturas locales de entorno.
- [ ] T031 [P] [US2] Cambiar constructores de `adapters/output/payment/mercadopago_provider.go` y `adapters/output/whatsapp/whatsapp_client.go` para recibir secretos/destinos tipados y cliente HTTP inyectado; eliminar lecturas locales de entorno.
- [ ] T032 [P] [US2] Implementar `adapters/output/ai/gemini_agent.go` como adaptador del puerto de modelo, con clave/modelo inyectados y error explícito de inicialización/ejecución en modo externo.
- [ ] T033 [US2] Eliminar lecturas de entorno y defaults silenciosos de `core/usecase/delivery_usecase.go`, `core/usecase/whatsapp_agent_usecase.go`, `adapters/input/http/handlers/whatsapp_handler.go` y `adapters/input/http/handlers/mercadopago_handler.go`; pasar reglas, tokens y puertos por constructor.
- [ ] T034 [US2] Seleccionar por modo independiente los cinco proveedores en `app/app.go`; ningún fallo de construcción en `test|live` puede cambiar a `mock`; registrar solo perfil/modos sin tokens, DSN, destinatarios o cuerpo.
- [ ] T035 [US2] Verificar en `config/validate.go` y `app/provider_identity.go` número WhatsApp de prueba y allowlist, cuenta MP de prueba y destino de notificación beta, y clave/proyecto Gemini beta antes de tráfico; rechazar identidad no verificable según `contracts/configuration.md`.
- [ ] T036 [US2] Restringir destinatarios y destinos de modo externo en `adapters/output/whatsapp/whatsapp_client.go`, `adapters/output/payment/mercadopago_provider.go` y `adapters/output/ai/gemini_agent.go`; un `*_BASE_URL` arbitrario no sustituye la identidad esperada.
- [ ] T037 [US2] Eliminar fallback que inventa respuestas cuando Gemini externo falla en `core/usecase/whatsapp_agent_usecase.go`; propagar error/controlar mensaje de indisponibilidad sin confirmar pago o envío.
- [ ] T038 [US2] Validar `PORT`, `DB_TRANSPORT` y secretos antes de registrar rutas en `app/app.go`; producir errores por campo y logs redactados de inicio.
- [ ] T039 [US2] Ejecutar `config/provider_modes_test.go` y `app/startup_test.go`, cubrir las ocho combinaciones test/mock y demostrar que activar un proveedor no activa otro.

**Checkpoint**: la misma versión arranca con selección explícita y no degrada silenciosamente un proveedor.

---

## Phase 5: User Story 3 — Promover la misma versión a producción (P2)

**Goal**: preparar el artefacto para Cloud Run con Cloud SQL, secretos externos, migración controlada y estado persistente.

**Independent Test**: usar la misma imagen con dos configuraciones, simular una migración fallida y comprobar que la nueva revisión no recibe tráfico ni migra en sus reinicios.

### Verificación primero

- [ ] T040 [P] [US3] Escribir `tests/production_startup_test.go` para arranque sin `.env`, `PORT` asignado, socket Unix, secretos externos y cero `AutoMigrate`/seed al reiniciar la API.
- [ ] T041 [P] [US3] Escribir `tests/deployment_gate_test.go` para paso de migración fallido que impide promoción y para repetición exitosa que conserva la revisión anterior hasta validación.

### Implementación

- [ ] T042 [US3] Completar `adapters/output/datasource/connection.go` con DSN MySQL de socket Unix `/cloudsql/project:region:instance` y TCP mutuamente excluyentes, comprobando conexión e identidad sin dirección local obligatoria.
- [ ] T043 [US3] Separar permisos y versiones de esquema en `cmd/migrate/main.go` y `adapters/output/datasource/migrations.go`: salida no cero ante fallo, seed administrativo solo en paso controlado y API de solo verificación.
- [ ] T044 [P] [US3] Definir `ports/output/file_store.go` y adaptadores `adapters/output/files/local_store.go` y `adapters/output/files/cloud_store.go`, agregando dependencia necesaria a `go.mod`, para archivos persistentes beta/Cloud Storage sin depender de `./uploads` en Cloud Run.
- [ ] T045 [US3] Reemplazar `app.Static` a rutas locales en `app/app.go` por el puerto de archivos y proteger el registro público en `adapters/input/http/routes/url_mappings.go` con operador/bootstrap; producción falla si carece de backend persistente.
- [ ] T046 [P] [US3] Crear `ports/output/event_inbox.go` y `adapters/output/repository/event_inbox_repository.go` con restricciones de `data-model.md`: `provider`, `externalEventID` «único», `profile`, `receivedAt`, `processingStatus`, `attempts` y referencia a payload protegido; guardar solo después de verificar firma y antes de confirmar webhook.
- [ ] T047 [US3] Crear `adapters/output/queue/` para entrega durable del evento (ejecutor local beta y Cloud Tasks autenticado en producción) y actualizar handlers de WhatsApp/MP/Uber en `adapters/input/http/handlers/` para verificar firma, guardar inbox antes del 200 y procesar sin goroutine posterior a la respuesta.
- [ ] T048 [US3] Persistir historial y deduplicación en `adapters/output/repository/conversation_repository.go` e inyectarlo en `adapters/input/http/session/session_manager.go`; usar `recipient`, historial «sin genai.Content», `updatedAt` y versión «para concurrencia», compartidos entre instancias y con retención/minimización documentadas.
- [ ] T049 [P] [US3] Crear `deploy/cloud-run/README.md` con imagen única, Secret Manager, cuenta/permiso separados de API y migración, Cloud SQL socket, Job de esquema, revisión `--no-traffic`, comprobaciones y promoción manual.
- [ ] T050 [US3] Ejecutar `tests/production_startup_test.go` y `tests/deployment_gate_test.go` más ensayo de varias instancias; confirmar archivos, sesiones y deduplicación persistentes sin migrar al arrancar.

**Checkpoint**: Cloud Run puede recibir la misma imagen tras un paso de esquema exitoso y sus instancias no mutan el esquema.

---

## Phase 6: User Story 4 — Conservar las reglas comerciales (P2)

**Goal**: probar conversación, conciliación y reparto con simuladores dinámicos persistentes y acciones explícitas del operador.

**Independent Test**: pedido beta pendiente → operaciones pendiente/rechazada/importe incorrecto sin pago → aprobada exacta conciliada → despacho manual → estados de envío válidos; cero llamadas reales.

### Verificación primero

- [ ] T051 [P] [US4] Escribir `adapters/input/http/handlers/beta_payment_handler_test.go` con contrato `POST/GET /api/beta/payments/operations`: 201/200/400/401/403/404/409, importe decimal textual de dos cifras, asociación a un solo pedido e idempotencia.
- [ ] T052 [P] [US4] Escribir `adapters/input/http/handlers/beta_delivery_handler_test.go` con contratos `POST /api/beta/deliveries/:external_delivery_id/transitions` y `GET /api/beta/deliveries/:external_delivery_id`, autenticación y rechazo posterior a estado final.
- [ ] T053 [P] [US4] Escribir `tests/beta_business_flow_test.go` para conciliación sin despacho automático, operador obligatorio, ocho modos externos y 100 sesiones mock con cero llamadas salientes.

### Modelos y adaptadores

- [ ] T054 [P] [US4] Crear `core/domain/beta_payment.go` y `adapters/output/repository/beta_payment_repository.go` con restricciones de `data-model.md`: `operationID` «string único, prefijo beta-pay-», «generado si no se envía; idempotencia por ID»; `orderID` «pedido beta existente y pendiente»; `amount` «positivo; comparación exacta con total del pedido, sin redondeo float nuevo»; `currency` «obligatorio» y `ARS`; `status` «explícito; sin valor predeterminado» entre `approved|pending|rejected`; `approvedAt` «solo para aprobada»; `createdBy` operador autenticado; `reconciledOrderID` «solo si conciliación dio MATCHED».
- [ ] T055 [P] [US4] Crear `core/domain/beta_delivery.go` y `adapters/output/repository/beta_delivery_repository.go` con restricciones de `data-model.md`: `quoteID` «prefijo beta-quote-», `orderID` «cuando la cotización se asigna al pedido», `addressID` y huella normalizada, distancia/tarifa/duración calculadas, `expiresAt` y `consumedAt` «consumo único»; `externalDeliveryID` «prefijo beta-del-», `orderID` «único», `trackingURL` «local beta», `status` `pending|en_route|delivered|cancelled` y `version` que «protege de dos transiciones concurrentes».
- [ ] T056 [US4] Implementar en `adapters/output/payment/mock_payment_provider.go` preferencias `beta-pref-` asociadas a pedido y operaciones persistentes; estado pendiente/rechazado no se considera pagado y `VerifyPaymentStatus` de ID desconocido nunca aprueba.
- [ ] T057 [US4] Corregir `core/usecase/payment_reconciliation_usecase.go` para conciliar una operación aprobada solo contra su `orderID`, importe exacto y ARS, impedir reutilización y propagar fallos de persistencia en una transacción; no despachar desde pago.
- [ ] T058 [US4] Crear `core/usecase/beta_payment_usecase.go` y `ports/input/beta_payment_port.go` para registro explícito por operador, ID idempotente y llamada a conciliación solo en `approved`.
- [ ] T059 [US4] Crear `adapters/input/http/handlers/beta_payment_handler.go` y registrar rutas beta en `adapters/input/http/routes/url_mappings.go` solo con MP mock y `StoreOperator`; responder con `simulated:true`, no afirmar acreditación real y `Cache-Control:no-store`.
- [ ] T060 [US4] Implementar en `adapters/output/delivery/mock_delivery_provider.go` fórmula determinista dependiente de distancia y destino validado; quote con ID/vencimiento, vínculo a dirección/pedido, rechazo de quote vencida/ajena y tracking beta local sin llamar a Uber.
- [ ] T061 [US4] Ajustar `core/usecase/delivery_usecase.go` para conservar despacho manual de pedido `PAID` con dirección válida y crear envío `pending` desde quote vigente; transición `pending→en_route|cancelled` y `en_route→delivered|cancelled`, sin cambios automáticos ni estados finales mutables.
- [ ] T062 [US4] Crear `core/usecase/beta_delivery_usecase.go` y `ports/input/beta_delivery_port.go` para transición iniciada por operador con bloqueo/versión, respuesta con pedido y envío, y actualización del pedido solo en `delivered|cancelled`.
- [ ] T063 [US4] Crear `adapters/input/http/handlers/beta_delivery_handler.go` y rutas beta de consulta/transición en `adapters/input/http/routes/url_mappings.go`, solo con Uber mock y `StoreOperator`; errores 400/401/403/404/409 del contrato.
- [ ] T064 [US4] Crear `adapters/input/http/handlers/beta_message_handler.go` y ruta `GET /api/beta/messages` con operador para inspeccionar captura persistente de `adapters/output/repository/mock_message_repository.go`, retención/redacción y `Cache-Control:no-store`.
- [ ] T065 [US4] Ajustar `adapters/output/ai/mock_agent.go` y `core/usecase/whatsapp_agent_usecase.go` para respuestas reproducibles de consulta de pago y entrega, etiquetadas como prueba; modo externo informa indisponibilidad sin inventar confirmaciones.
- [ ] T066 [US4] Endurecer `adapters/input/http/handlers/mercadopago_handler.go` y `adapters/input/http/handlers/uber_delivery_webhook_handler.go` para exigir firma/secretos en modos externos y no permitir que webhooks externos acrediten o cambien estado en modo mock.
- [ ] T067 [US4] Ejecutar tests de handlers y `tests/beta_business_flow_test.go`; verificar operaciones duplicadas, importe distinto, quote ajena/vencida, transiciones inválidas, despacho no autorizado y cero llamadas reales.
- [ ] T068 [US4] Actualizar `adapters/input/http/handlers/order_handler_test.go` y `core/usecase/delivery_usecase_test.go` para asegurar que el flujo existente solo despacha por operador con pedido pagado y no se dispara desde conciliación/agente.

**Checkpoint**: el operador controla cada pago y transición simulados; producción conserva proveedores reales y webhooks firmados.

---

## Phase 7: User Story 5 — Repetir la operación con una guía (P3)

**Goal**: otra persona puede ejecutar beta y preparar un despliegue sin editar código ni versionar secretos.

**Independent Test**: checkout limpio, instrucciones de `quickstart.md`, pedido beta completo en ≤15 minutos y revisión de guía Cloud Run.

- [ ] T069 [US5] Completar `docs/beta-local.md` con copia de `.env.beta.example`, inventario de recursos, comandos Compose, seed de datos ficticios, login, pedido/pago/despacho, retención del volumen y parada sin `-v`.
- [ ] T070 [P] [US5] Completar `deploy/cloud-run/README.md` con matriz de variables/secretos, DB/archivo persistente, validación de identidad, migración previa con rollback de tráfico y verificaciones posteriores sin ejecutar despliegue real.
- [ ] T071 [US5] Actualizar `README.md` con enlaces a `docs/beta-local.md`, `deploy/cloud-run/README.md` y contrato de configuración; retirar instrucciones obsoletas de `USE_MOCK_PROVIDERS` si existen.
- [ ] T072 [US5] Seguir desde un checkout limpio `specs/006-configurar-entornos-beta-produccion/quickstart.md`, anotar tiempos/resultados y corregir cualquier comando o prerequisito inexacto en esa guía.

**Checkpoint**: guía reproducible y sin secretos operativos.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T073 Ejecutar `go test ./...` y las pruebas de integración beta documentadas en `specs/006-configurar-entornos-beta-produccion/quickstart.md`; reparar regresiones concretas de pedidos, webhooks y autorización.
- [ ] T074 Verificar SC-002/008/009/010/011 mediante `tests/beta_isolation_test.go`, `tests/beta_business_flow_test.go` y `tests/deployment_gate_test.go`: diez ciclos aislados, 100 sesiones sin llamadas, estados válidos, ocho combinaciones y bloqueo por migración fallida.
- [ ] T075 [P] Revisar logs, imagen y archivos versionados con `docs/security-verification.md` para demostrar que secretos, DSN, cuerpos firmados y credenciales reales no aparecen; documentar límites de verificación de identidades externas.
- [ ] T076 Revisar trazabilidad FR-001–025 y SC-001–011 en `specs/006-configurar-entornos-beta-produccion/quickstart.md`; registrar evidencias y pendientes antes de considerar terminada la implementación.

## Dependencies & Execution Order

### Fases y grafo

```text
Phase 1 Setup → Phase 2 Foundational
                        ├→ US1 (beta aislada, MVP)
                        ├→ US2 (arranque seguro)
                        └→ US3 (Cloud Run)
US1 + US2 ────────────────→ US4 (flujos comerciales beta completos)
US1 + US2 + US3 + US4 ───→ US5 (guía final)
US1–US5 ─────────────────→ Polish
```

- **US1**: después de la base. Puede mostrar beta local aislada con simuladores básicos.
- **US2**: después de la base; puede avanzar en paralelo con US1 en archivos distintos, pero su validación final usa el Compose beta de US1.
- **US3**: después de la base y de la selección de modos de US2 para su ensayo final. Su diseño de migración es independiente de los controles beta de US4.
- **US4**: requiere DB/volumen de US1 y selección/identidades de US2; puede avanzar en paralelo con US3, serializando las ediciones coincidentes de `app/app.go` y `adapters/input/http/routes/url_mappings.go`.
- **US5**: documenta todos los incrementos terminados; necesita las cuatro historias previas para su recorrido completo.

### Orden dentro de historias

Las tareas de prueba de cada historia se escriben primero para fijar el comportamiento, luego modelos/puertos, adaptadores y casos de uso, después rutas y finalmente integración. Las tareas marcadas `[P]` pueden ejecutarse en paralelo solo tras sus prerrequisitos de fase; no indican permiso para editar el mismo archivo a la vez.

## Parallel Opportunities

- **US1**: T014/T015 pruebas independientes; T016/T017/T018 infraestructura en archivos distintos; T019 y T021 puertos/almacenamiento.
- **US2**: T028/T029 pruebas; T030/T031/T032 adaptadores de Uber/Maps, MP/WA y Gemini en archivos distintos.
- **US3**: T040/T041 pruebas; T044/T046/T049 archivos, inbox y guía en rutas distintas.
- **US4**: T051/T052/T053 pruebas; T054/T055 entidades/repositorios de pago y reparto en archivos distintos.
- **US5**: T069/T070 guías beta y Cloud Run en archivos distintos, una vez terminadas sus funciones.

## Implementation Strategy

1. **MVP**: completar Setup, Foundational y US1; verificar aislamiento y arranque mock sin tocar la operación real. US2 es la siguiente puerta de seguridad antes de habilitar cualquier modo `test` o ensayar producción.
2. Completar US2 y validar todas las combinaciones, errores de arranque e identidades.
3. Desarrollar US3 y US4 sobre esa base; validar cada historia por separado y luego juntas.
4. Cerrar US5 y Polish. El despliegue real y la migración de datos productivos quedan fuera del alcance del spec; el entregable es preparación y ensayo con recursos aislados.
