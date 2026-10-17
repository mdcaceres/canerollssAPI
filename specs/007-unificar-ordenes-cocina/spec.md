# Feature Specification: Órdenes unificadas para cocina y despacho

**Feature Branch**: `main` (sin rama nueva)

**Created**: 2026-10-17

**Status**: Draft

**Input**: Unificar órdenes Web/POS y Roli, reconocer ventas al verificarse el pago, actualizar cocina en tiempo real y solicitar Uber Moto por acción de cocina, con operación en beta y producción.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ver pedidos pagados en cocina (Priority: P1)

Como empleada de cocina, quiero que cada pedido confirmado aparezca con sabores, cantidades, cliente y dirección sin recargar la pantalla, para comenzar a prepararlo de inmediato.

**Why this priority**: La cocina no debe depender de una revisión manual para descubrir ventas cobradas.

**Independent Test**: Abrir dos tableros autorizados, confirmar un pago pendiente y comprobar que ambos muestran la misma comanda completa en menos de 200 ms; repetir tras una desconexión.

**Acceptance Scenarios**:

1. **Given** un pedido pendiente de Roli y dos tableros conectados, **When** la conciliación confirma su pago, **Then** aparece como pagado en ambos con cliente, sabores, cantidades y dirección.
2. **Given** un comprobante aún no conciliado, **When** un operador autorizado aprueba manualmente el pago, **Then** se usa la misma transición y se informa a cocina y al cliente.
3. **Given** un tablero desconectado durante nuevos pagos, **When** se reconecta, **Then** recupera la lista completa vigente antes de continuar con las novedades.

---

### User Story 2 - Registrar compras y caja al cobrar (Priority: P1)

Como dueño, quiero que los pedidos pendientes no aumenten compras ni ingresos cobrados, para que una transferencia abandonada no altere fidelización ni arqueo.

**Why this priority**: Corrige un error contable y comercial del flujo actual.

**Independent Test**: Crear un pedido pendiente de un cliente con dos compras, verificar que conserva dos y que la caja no registra cobro; confirmar y repetir la confirmación para verificar que queda en tres y el cobro se registra una sola vez.

**Acceptance Scenarios**:

1. **Given** un cliente con dos compras, **When** Roli crea un pedido pendiente, **Then** el contador permanece en dos y no se agrega ingreso cobrado.
2. **Given** ese pedido pendiente, **When** se verifica la transferencia, **Then** pasa a pagado, el contador pasa a tres y se registra un solo cobro.
3. **Given** un pago ya confirmado, **When** llegan confirmaciones repetidas o simultáneas, **Then** no hay un segundo incremento, cobro ni transición.
4. **Given** un pedido pendiente cancelado, **When** se consulta al cliente y a la caja, **Then** ninguno registra una compra o cobro por él.

---

### User Story 3 - Preparar y despachar cuando cocina lo decida (Priority: P1)

Como empleada de cocina, quiero marcar una comanda en preparación y pedir la moto cuando está lista, para evitar trabajo duplicado y una llegada prematura del repartidor.

**Why this priority**: El momento de despacho depende del trabajo real de cocina.

**Independent Test**: Marcar un pedido pagado de reparto en preparación, dejar vencer su cotización, solicitar despacho y verificar una sola moto, seguimiento guardado, actualización de tableros y aviso al cliente.

**Acceptance Scenarios**:

1. **Given** un pedido pagado, **When** cocina inicia su preparación, **Then** todos los tableros muestran que ya está siendo preparado.
2. **Given** un pedido de reparto pagado o en preparación con cotización vencida o ausente, **When** cocina solicita la moto, **Then** se obtiene una cotización vigente antes de pedirla.
3. **Given** un despacho aceptado, **When** se recibe su seguimiento, **Then** el pedido aparece despachado y el cliente recibe un mensaje de WhatsApp con el enlace.
4. **Given** dos solicitudes simultáneas de moto para un pedido, **When** se procesan, **Then** como máximo se crea un reparto externo; solicitudes posteriores muestran el despacho existente.
5. **Given** un pedido de retiro, no pagado, cancelado o ya despachado, **When** se intenta pedir moto, **Then** no se crea un reparto.

---

### User Story 4 - Crear ventas manuales con el mismo ciclo (Priority: P2)

Como operador de mostrador, quiero registrar una venta cobrada que requiere cocina, una venta entregada en el acto o una venta pendiente de transferencia, para usar las mismas reglas que Roli.

**Why this priority**: El canal manual no debe saltarse la preparación por omisión.

**Independent Test**: Crear una orden manual sin estado, una entregada en el acto y una pendiente; verificar estado inicial, visibilidad en cocina, fidelización y caja.

**Acceptance Scenarios**:

1. **Given** una orden manual sin estado elegido, **When** el operador la guarda, **Then** se considera pagada y aparece en cocina.
2. **Given** una venta cobrada y entregada en mostrador, **When** se registra como completada, **Then** cuenta como compra e ingreso una vez sin entrar a preparación.
3. **Given** una venta manual pendiente de transferencia, **When** se crea, **Then** queda sin compra ni cobro; al confirmarse, entra a cocina.

### Edge Cases

- Una pantalla lenta no retrasa pagos ni ventas. Si pierde novedades, la resincronización recupera el estado vigente.
- Un corte de red, cierre de sesión o reinicio obliga al tablero a reconstruir el estado; las novedades no constituyen un historial durable.
- Un pago rechazado, incompleto o de importe distinto no confirma la orden. Una aprobación manual requiere autorización y deja identificable su origen.
- Una orden sin cliente asociado no modifica estadísticas de cliente; las órdenes pagadas sí se cuentan en caja.
- El primer pago verificado fija la primera compra. Cancelar una orden pendiente no la fija.
- Se rechazan cambios operativos desde estados pendientes, cancelados o finales, y cambios incompatibles con retiro o reparto.
- Si falla la cotización o la creación del reparto, la orden conserva su estado y muestra el error.
- Si Uber acepta el despacho pero falla la respuesta o el guardado local, debe recuperarse el identificador externo antes de permitir otra solicitud.
- Si falla WhatsApp tras un despacho exitoso, el despacho permanece registrado, el seguimiento se muestra al operador y el aviso queda pendiente de reintento.
- Una cotización renovada puede tener otro precio: el pedido ya cobrado conserva su importe; el comercio asume la diferencia en esta entrega.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST aceptar solo los estados pendiente de pago, pagado, en preparación, listo para retiro, despachado, completado, entregado y cancelado, con transiciones válidas según modalidad.
- **FR-002**: Una orden pendiente MUST conservar ítems y datos de entrega, pero MUST dejar intactos compras del cliente, primera compra e ingresos cobrados de caja.
- **FR-003**: Una orden creada pagada o completada MUST registrar exactamente una compra y cobro; la pagada MUST entrar a cocina y la completada MUST quedar fuera de preparación.
- **FR-004**: Confirmar un pago MUST requerir orden pendiente y verificación externa o aprobación manual autorizada; MUST guardar referencia u origen y efectuar transición, fidelización y registro contable de forma consistente.
- **FR-005**: Confirmaciones repetidas y simultáneas MUST ser idempotentes respecto a estado, fidelización, caja y novedades.
- **FR-006**: Pantallas autorizadas MUST recibir avisos de creación, pago, cambios operativos, despacho y cancelación con comanda completa y hora local de Córdoba.
- **FR-007**: Las novedades MUST llegar sin bloquear ventas ante una pantalla lenta; una pantalla reconectada MUST poder consultar pedidos vigentes filtrados por uno o varios estados y reconstruir el tablero.
- **FR-008**: Consultar el tablero y recibir novedades MUST requerir sesión vigente y autorización sobre pedidos.
- **FR-009**: Los cambios operativos MUST validar estado de origen y modalidad; cada cambio válido MUST actualizar las pantallas conectadas.
- **FR-010**: Solo un pedido de reparto pagado o en preparación MUST poder solicitar moto; cotización ausente o vencida MUST renovarse antes de pedirla.
- **FR-011**: Cada pedido MUST producir como máximo un despacho externo, incluso ante solicitudes simultáneas o reintentos; identificador y seguimiento MUST guardarse antes de mostrarlo como despachado.
- **FR-012**: Tras despacho confirmado, el sistema MUST informar el seguimiento al cliente por WhatsApp y a cocina.
- **FR-013**: Una venta manual sin estado indicado MUST comenzar pagada; el operador MUST poder elegir pendiente, pagada o completada.
- **FR-014**: Beta y producción MUST conservar las mismas reglas de órdenes, pago y despacho, con dependencias propias de cada entorno y sin despacho automático por un pago.
- **FR-015**: El sistema MUST ofrecer un camino operativo para recuperar de forma segura un despacho aceptado por Uber cuando falle la respuesta o persistencia local, sin crear otra moto.
- **FR-016**: Los errores MUST distinguir transición inválida, pago ya confirmado, cotización fallida y despacho existente.

### Key Entities *(include if feature involves data)*

- **Orden**: Compra con modalidad, estado, ítems, importes, cliente, dirección y referencias externas.
- **Cliente**: Persona vinculada a órdenes, número de compras pagadas y fecha de primera compra pagada.
- **Caja**: Registro de ingresos efectivamente cobrados, vinculados a órdenes verificadas.
- **Cotización de entrega**: Precio, identificador, destino y vigencia de propuesta de reparto.
- **Despacho**: Solicitud única de repartidor, identificador externo y enlace de seguimiento.
- **Novedad de orden**: Cambio comercial u operativo observado por los tableros, con tipo, comanda completa y fecha/hora.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: En 95 de 100 pagos confirmados con tableros conectados, la comanda completa aparece en todas las pantallas de la misma instancia en menos de 200 ms.
- **SC-002**: En 100 pedidos pendientes abandonados o cancelados, el cambio en compras del cliente e ingresos cobrados es cero.
- **SC-003**: En 100 confirmaciones repetidas y 20 pares simultáneos, cada pedido produce exactamente una compra y un cobro.
- **SC-004**: En 20 pares de solicitudes simultáneas de moto, cada pedido produce como máximo un despacho externo.
- **SC-005**: Tras 20 interrupciones de conexión, todos los tableros reconstruyen la misma lista vigente que una consulta nueva.
- **SC-006**: En pruebas de reparto con cotización vencida, todas las solicitudes exitosas usan cotización vigente y muestran seguimiento a operador y cliente.
- **SC-007**: En una prueba guiada, al menos 9 de 10 operadores completan preparación y solicitud de moto sin ayuda ni duplicar una comanda.

## Assumptions

- La sesión y los roles actuales determinan quién ve pedidos y ejecuta acciones de cocina; la planificación verificará los permisos disponibles.
- Una venta manual sin estado explícito requiere preparación y se considera cobrada. Una completada representa entrega inmediata y cobro realizado.
- Las aprobaciones manuales generan una referencia auditable cuando no existe identificador externo.
- Caja registra solo cobros verificados. Si una orden pendiente se vinculó a una caja antes del pago, ese vínculo no se interpreta como ingreso y se reconoce en la caja abierta al confirmar.
- La cotización de reparto dura 15 minutos. Una nueva cotización no cambia el importe ya cobrado al cliente.
- Las novedades en vivo llegan a conexiones de una misma instancia. Para varias instancias hará falta distribución compartida; mientras tanto, cada entorno mantiene una instancia y los tableros resincronizan al reconectar.
- WhatsApp usa la integración existente. Credenciales y proveedores de beta y producción siguen las reglas de configuración del proyecto; esta función no activa servicios reales en beta por sí sola.
- Cambios de estado de entrega posteriores al despacho siguen las reglas existentes de la integración de reparto.
