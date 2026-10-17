# Contrato de configuración y arranque

Interfaz externa del proceso prevista para la implementación. Los nombres nuevos aquí son el contrato de esta función; los nombres antiguos de `.env.example` se migrarán de forma explícita. La configuración se carga una vez; un valor presente pero inválido siempre es error. El servicio no infiere `APP_ENV` ni convierte una credencial ausente en modo simulado.

## Variables comunes

| Variable | Tipo | Beta | Producción |
| --- | --- | --- | --- |
| `APP_ENV` | `beta|production` | obligatorio `beta` | obligatorio `production` |
| `PORT` | entero 1–65535 | valor local documentado, por defecto 8080 dentro del contenedor | inyectado por Cloud Run |
| `ENV_FILE` | ruta | opcional solo local; si se declara debe existir y ser válido | no se usa |
| `DB_TRANSPORT` | `tcp|unix` | `tcp` | `unix` para Cloud SQL |
| `DB_HOST`, `DB_PORT` | host, puerto | obligatorios con `tcp` | prohibidos con `unix` |
| `DB_SOCKET` | ruta absoluta `/cloudsql/project:region:instance` | prohibido con `tcp` | obligatorio con `unix` |
| `DB_NAME`, `DB_USER`, `DB_PASSWORD` | strings | usuario y DB exclusivos de beta | secretos/identidad de producción |
| `DB_EXPECTED_ID` | identidad de instancia o recurso | identidad beta fijada | identidad de Cloud SQL fijada |
| `JWT_SECRET` | secreto | distinto de producción | Secret Manager |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | usuario/secreto | solo para paso inicial beta | solo para paso controlado, no servicio |
| `AUTO_OPEN_REGISTER` | booleano estricto | opcional, valor documentado | opcional, por defecto falso |
| `STORE_NAME`, `STORE_ADDRESS`, `STORE_LAT`, `STORE_LNG`, `STORE_ZIP_CODE`, `STORE_PHONE_E164`, `DELIVERY_MAX_RADIUS_KM` | strings/números tipados | valores ficticios válidos | valores operativos requeridos |
| `FILES_BACKEND`, `FILES_LOCATION` | enum/ruta o bucket | directorio beta persistente | almacenamiento externo |

`DB_EXPECTED_ID` se compara con un marcador de entorno provisionado antes de migrar y con la identidad/alcance de la conexión, no solo con `DB_NAME`. En beta, el init del contenedor crea el marcador y un usuario restringido a esa base; la red Compose no conecta a la base real. En producción, el marcador se provisiona en un paso controlado previo. La cuenta de la API tiene privilegios menores que la cuenta de migración. Un marcador ausente o discrepante bloquea migración y servicio.

## Modos de proveedores

| Variable | Beta predeterminado | Beta opcional | Producción |
| --- | --- | --- | --- |
| `WHATSAPP_MODE` | `mock` | `test` | `live` |
| `GEMINI_MODE` | `mock` | `test` | `live` |
| `MP_MODE` | `mock` | `test` | `live` |
| `UBER_MODE` | `mock` | ninguno | `live` |
| `GEOCODING_MODE` | `mock` | ninguno en primera entrega | `live` |

En beta, una variable de modo ausente usa `mock`; en producción se exige `live` explícito para todas las salidas. `USE_MOCK_PROVIDERS` y flags de despacho automático dejan de ser controles de selección: si se suministran, la implementación debe rechazarlos con mensaje de migración de configuración, para evitar interpretaciones ambiguas. El cambio de modo requiere reiniciar y revalidar todo el proceso.

| Proveedor | Parámetros para `test` o `live` |
| --- | --- |
| WhatsApp | `WHATSAPP_VERIFY_TOKEN`, `WHATSAPP_API_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_APP_SECRET`; en test `WHATSAPP_TEST_PHONE_NUMBER_ID` esperado y `WHATSAPP_ALLOWED_RECIPIENTS` no vacío |
| Gemini | `GEMINI_API_KEY`, `GEMINI_MODEL`; en test credencial/proyecto beta y marcador `GEMINI_TEST_PROJECT_ID` verificado con la política de secretos |
| Mercado Pago | `MP_ACCESS_TOKEN`, `MP_WEBHOOK_SECRET`, `MP_ACCOUNT_ID` esperado; en test identidad/cuenta de prueba comprobada y URL de notificación beta |
| Uber | `UBER_DIRECT_CLIENT_ID`, `UBER_DIRECT_CLIENT_SECRET`, `UBER_DIRECT_CUSTOMER_ID`, `UBER_WEBHOOK_SECRET`, destinos API autorizados; solo `live` en producción |
| Geocodificación | `GOOGLE_MAPS_API_KEY`, destino/API autorizado en producción |

Los endpoints base de proveedor se fijan por adaptador o lista permitida según modo; un `*_BASE_URL` arbitrario no permite saltarse la validación de destino. URLs de prueba locales usadas en tests unitarios se inyectan por constructor de test, no mediante la configuración de servicio.

## Invariantes y errores

- `APP_ENV` ausente/desconocido, booleano o número mal formado, puertos fuera de rango, dos transportes de DB activos, credencial requerida faltante, identidad beta/productiva cruzada o modo prohibido: error antes de crear dependencias o aceptar HTTP.
- Un archivo opcional ausente no impide el arranque cuando no se declaró `ENV_FILE`; un `ENV_FILE` declarado e ilegible falla. Las variables del entorno prevalecen sobre valores del archivo local.
- Los errores nombran campos, nunca valores secretos. El log de arranque muestra solo perfil, transporte de DB y modos; no usuario, DSN, tokens, destinatarios ni cuerpos de mensaje.
- Un proveedor `test` que no pueda verificar su cuenta/identidad de prueba falla; no continúa en modo `mock`. Un proveedor `live` sin credenciales falla; no continúa en modo `mock`.
- `ADMIN_PASSWORD` no se acepta como un valor predeterminado en producción. El servicio no modifica usuarios administradores al iniciar.

## Contrato del proceso de migración

La misma imagen expone entrada separada `migrate` o un binario `cmd/migrate` con la misma carga/validación de perfil y DB. Tiene permisos de esquema que la API no necesita. Devuelve exit code 0 solo cuando esquema, versión y datos iniciales obligatorios se verificaron; cualquier fallo da exit code no cero y bloquea la promoción de la revisión. Debe ser seguro repetirlo. La API solo verifica compatibilidad de versión de esquema y conexión; nunca aplica cambios.
