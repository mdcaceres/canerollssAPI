# Implementation Plan: Entornos beta y producción aislados

**Branch**: `main` (directorio de función `006-configurar-entornos-beta-produccion`; no se crea rama en esta fase) | **Date**: 2026-10-17 | **Spec**: [spec.md](spec.md)

**Input**: especificación y cinco decisiones de `speckit-clarify`. Esta fase entrega diseño; `tasks.md` e implementación quedan para sus comandos respectivos.

## Summary

Preparar la misma aplicación Go para beta local en Docker y producción futura en Cloud Run mediante configuración tipada obligatoria, validación previa, selección independiente de adaptadores e inyección por constructor. Beta tendrá MySQL, volumen, credenciales y rutas de prueba exclusivos, con simuladores persistentes y control explícito del operador para pagos y entregas. La conexión, las migraciones y los datos iniciales se separan del inicio HTTP; producción obtiene secretos externamente y promociona la revisión solo después de un paso de esquema verificado.

## Technical Context

**Language/Version**: Go 1.26 (`go.mod`).  
**Primary Dependencies**: Fiber v2, GORM con driver MySQL; `google.golang.org/genai`, `godotenv`, JWT v5, validator. Se reutilizan las bibliotecas actuales; no se exige una nueva biblioteca de configuración.  
**Storage**: MySQL para beta y producción; SQLite solo para pruebas aisladas existentes. Cloud SQL MySQL previsto en producción por socket Unix; almacenamiento de archivos persistentes externo en Cloud Run.  
**Testing**: `go test ./...`, pruebas de configuración y composición, contratos HTTP de operador, integración MySQL beta, ensayos de fallos de arranque y despliegue. Las pruebas externas se activan explícitamente y usan identidades de prueba.  
**Target Platform**: Docker Compose local; contenedor Linux/amd64 en Google Cloud Run, con Cloud SQL y Secret Manager.  
**Project Type**: servicio HTTP/webhooks monolítico con arquitectura de puertos y adaptadores.  
**Performance Goals**: preservar la latencia y flujo actuales; arranque falla antes de aceptar tráfico ante configuración o DB inválida. No hay objetivo de throughput definido en la especificación.  
**Constraints**: `APP_ENV` explícito; beta sin acceso a recursos reales; cuatro salidas simuladas por defecto; tres integraciones de prueba independientes; despacho solo por operador; producción sin migraciones en instancia; datos que sobreviven al reinicio fuera del contenedor; secretos sin registro.  
**Scale/Scope**: una API, una DB por entorno, cinco integraciones externas (WhatsApp, Gemini, Mercado Pago, Uber, Maps), contratos beta de pago y entrega, preparación para varias instancias Cloud Run.

## Constitution Check

*GATE: evaluado antes de investigación y después del diseño.*

`.specify/memory/constitution.md` contiene solo marcadores de plantilla; no hay principios ratificados ni restricciones verificables. Por eso no se inventa una puerta constitucional. Se usan como puertas de esta función los requisitos ya aprobados de `spec.md`:

| Puerta | Antes de investigación | Tras diseño |
| --- | --- | --- |
| Configuración validada antes de dependencias/tráfico, sin fallback silencioso | Requiere diseño | Cubierta por `config.Load/Validate` y composición que devuelve error |
| Aislamiento de beta y cuatro proveedores simulados por defecto | Requiere diseño | Cubierta por identidades, Compose exclusivo y matriz de modos |
| Pagos/entregas beta solo por acción autorizada | Requiere diseño | Cubierta por puertos, estado persistente y contratos `/api/beta/*` |
| Despacho manual, conciliación y firmas preservadas | Requiere diseño | Cubierta por reutilización de casos de uso y rutas firmadas separadas |
| Producción sin migración por instancia y con secretos externos | Requiere diseño | Cubierta por comando de migración, Job y promoción sin tráfico |

**Resultado**: sin violaciones constitucionales conocidas ni `NEEDS CLARIFICATION` pendientes. La ausencia de una constitución ratificada no autoriza a cambiar el alcance de la especificación.

## Architecture and Dependency Order

La organización actual `core`, `ports`, `adapters`, `app` es aprovechable. El ajuste necesario es el sentido de las dependencias y el punto en que se selecciona cada proveedor:

```text
main.go ──→ app/ ──→ config/
              ├──→ adapters/input/http y adapters/output/* ──→ ports/*
              └──→ core/usecase ──→ ports/* y core/domain
cmd/migrate ──→ config/ y adapters/output/datasource (sin app HTTP)
```

Las flechas indican dependencias de importación o construcción: los adaptadores implementan puertos y pueden importar sus tipos; `core/usecase` importa puertos y dominio; `app` conoce implementaciones. `config` no entra en dominio ni en casos de uso: `app` pasa valores concretos a constructores. `ports` no importa Fiber, GORM ni el SDK Gemini. El acoplamiento GORM ya existente en algunas entidades se registra como deuda fuera del cambio general; las nuevas entidades de simulación no lo amplían al dominio.

**Secuencia de arranque de la API**:

1. Cargar archivo local solo si se solicitó o se usa la convención local documentada; variables del proceso prevalecen. Parsear y validar campos, perfil, modos, credenciales, puertos, identidades y combinaciones.
2. Confirmar que el destino de DB corresponde al perfil con identidad/usuario y permisos separados; abrir y hacer ping a la DB sin migrarla.
3. Construir repositorios, almacenes beta y cada adaptador seleccionado; verificar identidad de proveedores externos de prueba cuando haya modo `test` y no admitir fallback.
4. Inyectar puertos y parámetros de negocio en casos de uso; construir middleware y handlers con secretos del perfil. Registrar controles beta solo en beta y modo simulado correspondiente; no registrar rutas de simulación abiertas en producción. Los webhooks externos verifican firma, guardan eventos con clave idempotente y delegan trabajo largo a un disparador durable; historial y deduplicación se comparten entre instancias.
5. Registrar perfil y modos sin credenciales; iniciar Fiber en `0.0.0.0:$PORT`. Errores de los pasos previos detienen el proceso.

**Secuencia de despliegue**: crear nueva imagen inmutable → revisar esquema compatible con revisión activa → ejecutar migración/seed separada y verificar resultado → desplegar revisión Cloud Run sin tráfico → validar salud y conexiones → transferir tráfico. Un fallo de migración o validación conserva la revisión anterior. Los cambios de esquema incompatibles exigen estrategia expandir/contraer; `AutoMigrate` no se considera rollback.

## Project Structure

### Documentation (this feature)

```text
specs/006-configurar-entornos-beta-produccion/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── contracts/
│   ├── configuration.md
│   └── beta-http.md
├── quickstart.md
└── tasks.md                 # se generará con speckit-tasks
```

### Source Code (repository root; estructura objetivo, no creada aún)

```text
main.go                      # arranque HTTP, error fatal sin secretos
cmd/migrate/                # entrada separada para esquema/seed controlado
config/                     # carga, structs, validación y matriz de modos
app/                        # composición, ciclo de vida, selección de adaptadores
core/domain/                # pedidos y tipos de estado independientes de SDK
core/usecase/               # reglas de pago, reparto, conversación y operación beta
ports/input/                # casos de uso invocados desde HTTP
ports/output/               # proveedores, almacenes y modelo conversacional
adapters/input/http/
├── handlers/                # controles beta y webhooks separados
├── middleware/              # autenticación con secreto inyectado
├── routes/                  # registro de rutas según perfil validado
└── session/                 # estado conversacional; sin tipos Gemini en contrato
adapters/output/
├── datasource/              # apertura y verificación de DB, sin migrar
├── repository/              # pedidos y estados persistentes de beta
├── payment/                 # Mercado Pago real/test y simulador
├── delivery/                # Uber real y simulador de cotización/estado
├── geocoding/               # Maps real/test y simulador
├── whatsapp/                # envío real/test y captura beta
├── ai/                      # Gemini real/test y agente reproducible beta
├── queue/                   # entrega durable de eventos externos
└── files/                   # almacenamiento persistente local/cloud
Dockerfile
compose.beta.yml             # API + MySQL beta con proyecto/volumen propios
.env.beta.example            # valores ficticios sin secretos
docs/                        # operación beta y preparación de Cloud Run
tests/                       # escenarios de integración actuales y nuevos
```

**Structure Decision**: se conservan los paquetes existentes y se agregan solo los bordes necesarios. No se introduce un contenedor de DI ni una carpeta `internal/` paralela. El control de pruebas pertenece a `adapters/input/http`, no a proveedores reales ni a un webhook público.

## Design Traceability

| Requisito | Diseño | Validación principal |
| --- | --- | --- |
| FR-001–006, 017, 025 | `config` y `app`; contrato de variables | matriz de parseo, fallos y logs redactados |
| FR-007–010, 019–020 | selección individual y DB/volumen beta | 8 combinaciones, aislamiento TCP y credenciales |
| FR-011–013, 018 | imagen, socket Cloud SQL, Secret Manager, migración y archivos | arranque sin `.env`, reinicios, ensayo de fallo de migración |
| FR-014–015, 021–024 | puertos, simuladores persistentes, contratos HTTP beta | pago pendiente/rechazado/aprobado, cotización, despacho y estados |
| FR-016 | `quickstart.md` y guía operativa en implementación | recorrido desde checkout limpio |

## Complexity Tracking

No hay violaciones de constitución ratificada. La persistencia de los simuladores y el adaptador de modelo conversacional añaden componentes porque los controles de operador y el arranque seguro deben funcionar tras reinicios y con selección independiente; las alternativas en memoria o con SDK dentro del caso de uso incumplen FR-019/021–024.
