# Feature Specification: Entornos beta y producción aislados

**Feature Branch**: `sin rama nueva (directorio 006)`

**Created**: 2026-10-17

**Status**: Draft

**Input**: Arquitectura de entornos 12-Factor con configuración tipada e inyección de dependencias para Canerollss; beta local en contenedores y despliegue posterior en Google Cloud Run.

## Clarifications

### Session 2026-10-17

- Q: Cuando probemos un pedido en beta, ¿cómo debería pasar su pago ficticio a “aprobado”? → A: Un operador autorizado registra explícitamente una operación ficticia; el sistema la concilia contra el pedido. Debe permitir probar operaciones aprobadas, pendientes y rechazadas sin acreditación automática.
- Q: Después de despachar manualmente un pedido en beta, ¿cómo deberían cambiar los estados del envío simulado? → A: Un operador autorizado cambia explícitamente el estado simulado entre pendiente, en camino, entregado y cancelado; no hay transiciones automáticas por tiempo.
- Q: Además de los simuladores por defecto, ¿qué conexiones externas de prueba se necesitan en beta desde la primera entrega? → A: WhatsApp de prueba, Gemini y Mercado Pago de prueba, cada uno activable por separado; los demás permanecen simulados.
- Q: Para ejecutar beta junto al servicio real en la máquina local, ¿dónde deben guardarse los datos de prueba? → A: En un contenedor de base de datos y un volumen exclusivos de beta, independientes de los recursos del servicio real.
- Q: Al desplegar una nueva versión en producción, ¿cuándo deben aplicarse los cambios de esquema y datos iniciales? → A: En un paso de despliegue separado y controlado, completado y verificado antes de activar la nueva versión para recibir tráfico.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Probar beta sin afectar ventas (Priority: P1)

Como desarrollador, quiero iniciar y detener beta mientras el servicio que atiende pedidos reales permanece activo, para probar pedidos, pagos y entregas sin tocar datos ni cuentas operativas reales.

**Why this priority**: La separación protege la operación comercial actual.

**Independent Test**: Iniciar beta con datos de prueba, crear un pedido, detener beta y verificar que el servicio real sigue disponible y sus datos no cambiaron.

**Acceptance Scenarios**:

1. **Given** el servicio real activo, **When** inicia beta local, **Then** ambos son accesibles simultáneamente en direcciones diferentes; beta usa su propio contenedor de base de datos y volumen, sin montar ni conectarse a los recursos de datos reales.
2. **Given** beta activo, **When** se crea o elimina un pedido de prueba, **Then** no cambia ningún dato real ni se llama a cuentas operativas de pago, geocodificación o reparto.
3. **Given** beta activo, **When** se detiene o reinicia, **Then** el servicio real continúa atendiendo solicitudes y los datos beta siguen la política de persistencia documentada.

---

### User Story 2 - Arrancar solo con configuración segura (Priority: P1)

Como operador, quiero conocer antes de aceptar tráfico si faltan credenciales, si un valor es inválido o si se ha seleccionado el recurso equivocado, para evitar operaciones incorrectas.

**Why this priority**: Una selección silenciosa de proveedores simulados o de una base equivocada puede afectar ventas y despachos.

**Independent Test**: Arrancar con perfiles válidos y luego quitar o alterar, uno por uno, campos críticos; verificar aceptación o rechazo explícito sin imprimir secretos.

**Acceptance Scenarios**:

1. **Given** configuración beta válida, **When** inicia, **Then** se seleccionan dependencias de prueba y se informa el perfil sin revelar secretos.
2. **Given** producción sin una clave, credencial o destino obligatorio, **When** inicia, **Then** se detiene antes de aceptar solicitudes con un error que identifica el campo.
3. **Given** un booleano, número, puerto o perfil mal formado, **When** inicia, **Then** se rechaza el valor y no se cambia silenciosamente a un proveedor alternativo.
4. **Given** configuración suministrada por el entorno y ningún archivo local, **When** inicia, **Then** funciona sin exigir un archivo dentro del contenedor.

---

### User Story 3 - Promover la misma versión a producción (Priority: P2)

Como responsable del despliegue, quiero usar la misma versión en beta y producción, cambiando únicamente la configuración externa, para desplegar después en Google Cloud Run sin reescribir la aplicación.

**Why this priority**: Reduce diferencias entre lo probado y lo publicado.

**Independent Test**: Ejecutar la misma versión con perfiles diferentes; comprobar inicio, destino de datos, puerto asignado y obtención externa de secretos.

**Acceptance Scenarios**:

1. **Given** una versión preparada, **When** se ejecuta en beta o producción, **Then** usa el mismo artefacto y las dependencias previstas para cada perfil.
2. **Given** un puerto asignado por la plataforma, **When** inicia el servicio, **Then** escucha allí y es accesible desde fuera del contenedor.
3. **Given** un almacenamiento administrado autorizado, **When** se configura su conexión, **Then** el servicio accede a la base prevista sin depender de una dirección local.
4. **Given** un secreto rotado, **When** se inicia una nueva revisión, **Then** se recibe desde la configuración externa sin reconstruir el artefacto.
5. **Given** una nueva versión que requiere cambios de esquema o datos iniciales, **When** se despliega a producción, **Then** se ejecutan y verifican en un paso controlado antes de activar la versión; un fallo impide la activación y ninguna instancia aplica esos cambios al arrancar.

---

### User Story 4 - Conservar las reglas comerciales (Priority: P2)

Como operador de la tienda, quiero que el cambio de entornos conserve los flujos de WhatsApp, pedidos, conciliación y entregas, para que una mejora operativa no altere quién puede despachar.

**Why this priority**: La configuración toca todos los proveedores y puede introducir regresiones de negocio.

**Independent Test**: Ejecutar los flujos actuales en beta con proveedores simulados; verificar que el despacho sigue requiriendo una acción del operador y un pedido pagado.

**Acceptance Scenarios**:

1. **Given** un pedido beta pagado, **When** se concilia el pago, **Then** no se solicita automáticamente un repartidor.
2. **Given** un pedido beta pendiente de pago, **When** un operador autorizado registra una operación ficticia aprobada por el importe correcto, **Then** el sistema la concilia y el pedido pasa a pagado; una operación pendiente, rechazada o de importe distinto no lo paga.
3. **Given** un pedido válido y pagado, **When** un operador autorizado solicita el despacho, **Then** se usa el proveedor del entorno y se conservan las validaciones de pedido y dirección.
4. **Given** un envío beta ya despachado, **When** un operador autorizado cambia su estado simulado a en camino, entregado o cancelado, **Then** el pedido refleja la transición válida y una transición inválida o posterior a un estado final se rechaza sin llamar a Uber.
5. **Given** beta sin credenciales externas, **When** inicia y se prueba el flujo conversacional, **Then** WhatsApp, Gemini, Mercado Pago y Uber usan sustitutos de prueba y no envían mensajes ni generan cargos o despachos reales.
6. **Given** beta con WhatsApp, Gemini o Mercado Pago de prueba activado individualmente, **When** inicia, **Then** valida las credenciales y el destino beta de ese proveedor antes de aceptar solicitudes y mantiene simulados los proveedores no activados.
7. **Given** una respuesta simulada de pago, **When** el cliente pregunta si pagó, **Then** la respuesta identifica el resultado como prueba y nunca afirma que el dinero real fue acreditado.
8. **Given** producción con un proveedor real seleccionado pero sin credencial, **When** inicia, **Then** no lo sustituye silenciosamente por uno simulado.

---

### User Story 5 - Repetir la operación con una guía (Priority: P3)

Como responsable técnico, quiero instrucciones para iniciar beta, revisar su estado y preparar el despliegue, para repetir pruebas sin utilizar secretos reales.

**Why this priority**: La operación segura necesita un procedimiento transferible.

**Independent Test**: Una persona ajena a la implementación sigue la guía desde un checkout limpio y completa una prueba de pedido beta.

**Acceptance Scenarios**:

1. **Given** un checkout limpio y valores de prueba, **When** se sigue la guía, **Then** beta y su almacenamiento arrancan sin editar código ni versionar credenciales.
2. **Given** la guía de despliegue, **When** se revisa, **Then** enumera configuración, secretos, conexión de datos y comprobaciones posteriores.

### Edge Cases

- Beta recibe por error la misma identidad o credenciales de base que producción: se rechaza antes de escribir; el nombre de la base por sí solo no basta para aislar.
- Falta una opción opcional: se aplica un valor documentado. Falta una opción obligatoria: se rechaza el arranque.
- Se indican dos mecanismos de conexión incompatibles: se informa el conflicto y se rechaza el arranque.
- Falta un archivo local opcional: la configuración externa sigue funcionando. Un archivo local indicado expresamente y mal formado produce un error visible.
- Fallan la conexión de datos o su preparación: el servicio no acepta pedidos con inicialización parcial. Si falla el paso controlado de cambios de producción, la versión anterior permanece activa.
- Una integración está deshabilitada: el flujo informa indisponibilidad; no simula una confirmación real de pago, envío o mensaje.
- Una credencial real aparece accidentalmente en la configuración beta mientras el proveedor está en modo simulado: no se usa para llamadas salientes ni se muestra en registros.
- Un proveedor beta se configura como real o de prueba externa sin destino y credenciales beta verificables: el arranque se rechaza; no se prueba contra la cuenta productiva.
- La plataforma reinicia o escala una instancia: datos de negocio y archivos que deban persistir permanecen fuera del almacenamiento temporal de la instancia.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST reconocer solo los perfiles `beta` y `production`; un perfil ausente o desconocido MUST rechazar el arranque antes de crear dependencias o aceptar tráfico.
- **FR-002**: El sistema MUST leer y validar la configuración operativa antes de construir dependencias o aceptar solicitudes, con valores tipados y errores por campo.
- **FR-003**: El sistema MUST aceptar configuración externa sin exigir archivos locales dentro del artefacto; los archivos de desarrollo MUST ser opcionales y excluir secretos reales del repositorio.
- **FR-004**: El sistema MUST ejecutar la misma versión en beta y producción; el destino MUST elegirse por configuración, sin alterar reglas de negocio.
- **FR-005**: El sistema MUST entregar configuración validada y dependencias al iniciar cada componente, preservando la separación entre reglas de negocio, interfaces de proveedores y adaptadores; esos componentes no deben consultar por su cuenta la configuración del proceso.
- **FR-006**: El sistema MUST cubrir las opciones existentes de almacenamiento, autenticación, usuario administrador, WhatsApp, Gemini, Mercado Pago, Google Maps, Uber Direct, tienda, radio de reparto y operación.
- **FR-007**: El sistema MUST seleccionar explícitamente proveedores reales o simulados de geocodificación, pago y reparto; producción MUST rechazar proveedores simulados o credenciales incompletas.
- **FR-008**: El sistema MUST impedir que beta utilice recursos persistentes de producción mediante identidades y permisos separados y validación antes de operaciones de escritura.
- **FR-009**: El sistema MUST permitir el inicio y ciclo de vida de beta local de forma independiente y simultánea al servicio real. En la máquina local, beta MUST usar un contenedor de base de datos y un volumen exclusivos, además de una dirección de acceso distinta; detener o eliminar beta no MUST afectar recursos del servicio real.
- **FR-010**: El sistema MUST permitir conexión local y conexión al almacenamiento administrado de producción sin cambiar la lógica de negocio.
- **FR-011**: El sistema MUST escuchar en el puerto asignado por el entorno y ser accesible desde fuera del contenedor.
- **FR-012**: El sistema MUST mantener secretos fuera del artefacto y los registros; los secretos de producción MUST suministrarse mediante un gestor externo.
- **FR-013**: El sistema MUST detenerse antes de aceptar tráfico si falla la conexión de datos o la preparación obligatoria. En producción, cambios de esquema y datos iniciales MUST ejecutarse y verificarse en un paso de despliegue separado, antes de activar la nueva versión; las instancias de servicio no MUST aplicar migraciones ni modificar credenciales administrativas al arrancar. Un fallo de ese paso MUST impedir la activación.
- **FR-014**: El sistema MUST conservar el despacho manual de Uber por operador autorizado para pedidos pagados y válidos; ninguna opción de entorno MUST habilitar despacho automático desde pagos o el agente.
- **FR-015**: El sistema MUST conservar conciliación, verificación de webhooks y autenticación con secretos correspondientes al perfil activo.
- **FR-016**: El proyecto MUST proporcionar instrucciones reproducibles para iniciar, comprobar y detener beta, preparar datos de prueba y verificar un despliegue de producción.
- **FR-017**: El sistema MUST identificar en registros no sensibles el perfil y tipo de proveedor activos para verificar el destino de cada instancia.
- **FR-018**: Los datos de negocio y archivos que deban sobrevivir al reinicio MUST conservarse en almacenamiento persistente separado por entorno.
- **FR-019**: Beta MUST iniciar por defecto con cuatro dependencias salientes de prueba independientes: mensajería WhatsApp, agente Gemini, pagos Mercado Pago y reparto Uber; no basta un interruptor global que cubra solo algunas de ellas.
- **FR-020**: Desde la primera entrega, beta MUST poder habilitar individualmente integraciones externas de prueba para WhatsApp, Gemini y Mercado Pago mediante elección explícita, credenciales separadas y destino beta validado; habilitar uno no MUST habilitar los otros. Uber MUST permanecer simulado hasta que exista y se valide un destino de pruebas separado del operativo.
- **FR-021**: El modo WhatsApp simulado MUST aceptar conversaciones de prueba y registrar mensajes salientes de forma segura sin enviarlos a clientes. El modo externo de prueba MUST limitar los destinatarios a números autorizados y usar identidad y token distintos de producción.
- **FR-022**: El modo Gemini simulado MUST ofrecer respuestas de prueba reproducibles para los flujos cubiertos sin llamadas externas. El modo externo de prueba MUST usar una clave beta separada y datos ficticios; un fallo de inicialización MUST impedir ese modo, sin degradar silenciosamente a una respuesta simulada.
- **FR-023**: El modo Mercado Pago simulado MUST generar preferencias y operaciones ficticias identificables por pedido, con estado de pago variable. Solo un operador autorizado MUST poder registrar explícitamente una operación ficticia aprobada, pendiente o rechazada; únicamente una aprobada y conciliada por el importe correcto puede marcar el pedido como pagado. No habrá acreditación automática al crear un pedido. El modo externo de prueba MUST usar credenciales y notificaciones de prueba separadas; ningún modo simulado MUST declarar acreditado un pago sin una transacción de prueba verificada.
- **FR-024**: El modo Uber simulado MUST producir cotizaciones ficticias con identificador y vencimiento, cuyo importe y duración estimada varían según la distancia al destino validado; un despacho ficticio MUST corresponder a una cotización vigente para la misma dirección validada y el pedido que se despacha. El despacho sigue siendo manual. Tras él, solo un operador autorizado MUST poder avanzar explícitamente el estado simulado entre pendiente, en camino, entregado y cancelado, respetando transiciones válidas y estados finales; no habrá cambios automáticos por tiempo ni llamadas a Uber. Producción MUST usar credenciales reales válidas y conservar el control de autorización y firmas de notificaciones.
- **FR-025**: El inicio MUST registrar únicamente el perfil y el modo seleccionado de cada proveedor; cualquier combinación beta/producción que pueda contactar una cuenta operativa desde beta MUST rechazarse antes de aceptar tráfico.

### Política de proveedores al iniciar

| Dependencia | Beta por defecto | Activación externa en beta | Producción |
| --- | --- | --- | --- |
| Uber Direct | Cotización y despacho ficticios; el operador cambia explícitamente el estado posterior | Deshabilitada hasta verificar un destino de pruebas separado | Proveedor real con credenciales válidas |
| Mercado Pago | Preferencias y operaciones ficticias con estados variables; un operador registra el pago y activa su conciliación | Solo cuenta y credenciales de prueba, activadas explícitamente | Proveedor real; nunca fallback a simulación |
| WhatsApp | Entrada de prueba y salida capturada sin envío | Solo número de prueba, token beta y destinatarios autorizados | Envío y verificación reales con secretos válidos |
| Gemini | Respuestas reproducibles de prueba, sin llamada externa | Clave beta y datos ficticios, activados explícitamente | Servicio real con clave válida; sin fallback que invente confirmaciones |

La selección se realiza una sola vez antes de aceptar solicitudes y se entrega a los casos de uso mediante sus dependencias. La falta de credenciales de un modo explícitamente activado produce un error de inicio; nunca cambia de modo automáticamente.

### Key Entities *(include if feature involves data)*

- **Perfil de entorno**: Identifica beta o producción y las políticas aplicables.
- **Configuración validada**: Valores operativos tipados de puerto, datos, proveedores, autenticación y reglas.
- **Identidad de recurso**: Base, credenciales y destino de proveedor asociados a un entorno.
- **Secreto**: Valor confidencial externo al artefacto, con destino y propósito definidos.
- **Instancia del servicio**: Ejecución concreta de una versión con un perfil y configuración definidos al arrancar.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Una persona con checkout limpio y valores de prueba inicia beta y verifica un pedido en 15 minutos o menos siguiendo la documentación.
- **SC-002**: En 10 ciclos de inicio, parada y reinicio de beta con su contenedor y volumen de datos exclusivos, el servicio real conserva el 100 % de disponibilidad durante las comprobaciones y sus datos registran 0 cambios causados por beta.
- **SC-003**: El 100 % de configuraciones de producción probadas con campos críticos faltantes o inválidos se rechazan antes de aceptar solicitudes.
- **SC-004**: En pruebas beta con proveedores simulados se registran 0 llamadas a pagos, geocodificación y reparto reales.
- **SC-005**: La misma versión supera el inicio y el flujo de pedido en beta y en un entorno de preproducción configurado como producción, sin cambios de código.
- **SC-006**: El 100 % de secretos del ensayo de despliegue se suministran fuera del artefacto y no aparecen en registros.
- **SC-007**: El 100 % de pruebas de conciliación conserva la aprobación explícita por operador: pagos pendientes, rechazados o de importe incorrecto dejan el pedido sin pagar; el 100 % de pruebas de despacho mantiene el despacho manual y rechaza pedidos no pagados.
- **SC-008**: Con la configuración beta predeterminada, 100 sesiones de prueba generan 0 llamadas externas de WhatsApp, Gemini, Mercado Pago y Uber, y 0 mensajes que afirmen una acreditación real sin verificación.
- **SC-009**: El 100 % de pruebas de estados de envío beta aceptan cambios válidos iniciados por operador y rechazan cambios automáticos, no autorizados o posteriores a un estado final.
- **SC-010**: Las 8 combinaciones de activación individual de WhatsApp, Gemini y Mercado Pago de prueba seleccionan solo los proveedores pedidos y rechazan cualquier combinación con credenciales o destinos beta incompletos.
- **SC-011**: En el 100 % de ensayos de despliegue con un cambio de esquema fallido, la nueva versión no recibe tráfico; en 10 reinicios de instancias de producción, ninguna ejecuta cambios de esquema ni datos iniciales.

## Assumptions

- El alcance incluye beta local y preparación verificable para Cloud Run; no incluye desplegar ni migrar datos reales durante esta fase.
- Puede existir un servicio real activo en la máquina. Antes de ejecutar ambos se inventariarán sus puertos, datos y credenciales; hoy no existe una configuración de contenedores en el repositorio.
- La operación prevista utiliza MySQL y las pruebas existentes usan SQLite aislado. Beta local tendrá su propio contenedor y volumen de datos, con identidad y permisos sin acceso de escritura a producción.
- Ya existen integraciones con WhatsApp, Gemini, Mercado Pago, Google Maps y Uber Direct. La activación real de cada una requiere selección explícita y validación completa; beta utiliza simulación para las cuatro integraciones consultadas y para Google Maps por defecto.
- Se mantiene la decisión existente de despacho manual por operador; la opción reservada de despacho automático permanece inactiva.
- Producción requiere credenciales únicas y seguras, sin contraseñas administrativas predeterminadas; beta puede usar datos y credenciales ficticios.
- La planificación incluirá el paso controlado de migraciones y datos iniciales previo a activar producción, además del tratamiento de archivos persistentes; contenerizar el servicio no resuelve esos aspectos por sí solo.
- La documentación adjunta orienta la arquitectura, pero sus afirmaciones de costo, cuota y tiempo de arranque no son criterios de aceptación.
