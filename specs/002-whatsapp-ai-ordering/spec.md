# Feature Specification: WhatsApp AI-Powered Order Automation & Transactional Flow

**Feature Branch**: `002-whatsapp-ai-ordering`  
**Created**: 2026-10-16  
**Last Updated**: 2026-10-16  
**Status**: Ready for Planning (`/speckit.plan`)  
**Input**: Extension of WhatsApp cinnamon rolls sales agent to support 1:N customer address management, automatic pre-order delivery quotation via Uber Direct port, order creation with `PENDING_PAYMENT` status and Mercado Pago payment link generation, and payment status verification. Customers never type their phone number; it is extracted securely from the WhatsApp session (`wa_id`) and persisted by the backend.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Customer Builds and Confirms an Order via WhatsApp (Priority: P1)

As a customer of the cinnamon roll store, I want to message the business WhatsApp number with what I'd like to order (e.g., "Quiero 2 combos de Nutella"), define whether I want pickup or delivery, review the itemized breakdown (products + delivery cost) with total price, and explicitly confirm before the order is created in the database and a payment link is issued.

**Why this priority**: This is the core revenue-generating capability. In order to avoid creating premature, abandoned, or incomplete orders, the agent must NOT create an order in the database on the very first message. The agent must first confirm delivery modality (pickup or delivery), quote delivery if applicable, show the itemized total, and only upon the customer's confirmation create the order in `PENDING_PAYMENT` status.

**Independent Test**: Send a WhatsApp message ordering items. Verify the agent asks or confirms delivery modality (pickup or delivery) and displays the itemized price breakdown. Then, send an explicit confirmation (e.g., "Confirmar pedido", "Sí, dale"). Verify the order is created in the database with status `PENDING_PAYMENT`, customer linked by `wa_id`, and a Mercado Pago payment link is returned.

**Acceptance Scenarios**:

1. **Given** the store has an open cash register and active flavors in the catalog, **When** a customer sends "Quiero 2 combos de Nutella", **Then** the agent identifies the flavor, quantity, and format, asks if it is for pickup or delivery, and presents the subtotal without creating an order in the database yet.
2. **Given** a customer who selected items and delivery modality (e.g., "Retiro por el local"), **When** the customer sends "Confirmo el pedido", **Then** the system registers the order with `status = PENDING_PAYMENT`, `delivery_type = PICKUP`, `delivery_cost = 0`, generates the Mercado Pago payment link, and sends the order summary with payment link.
3. **Given** a customer ordering multiple flavors in a single request (e.g., "1 combo de Nutella y 2 individuales de Pistacho premium"), **When** the agent processes the request, **Then** it properly builds multiple line items in the order breakdown.
4. **Given** a new customer (phone number not yet registered), **When** their order is confirmed, **Then** the system automatically creates a customer record using their WhatsApp phone number (`wa_id`) and profile name, without ever asking the customer to type their phone number.
5. **Given** a returning customer, **When** their order is confirmed, **Then** the system associates the order with their existing customer profile.

---

### User Story 2 - Customer Inquires About the Menu and Prices (Priority: P2)

As a customer, I want to ask "What flavors do you have?" or "How much is a combo?" via WhatsApp and receive an accurate, up-to-date list of available flavors with their current prices for both individual and combo formats.

**Why this priority**: Customers need to know what's available and how much it costs before placing an order. This is the natural precursor to ordering and reduces abandoned conversations.

**Independent Test**: Send a message asking about the menu. Verify the system queries the current active toppings/flavors from the database and responds with accurate names and prices, using natural flavor terminology rather than technical terms like "topping".

**Acceptance Scenarios**:

1. **Given** the catalog has active flavors with current prices, **When** a customer sends "What flavors do you have?", **Then** the AI responds with a formatted list of all active flavors showing name, individual price, and combo price, referring to them as "sabores" or "roles", never "toppings".
2. **Given** the catalog has one flavor marked as inactive, **When** a customer asks about the menu, **Then** the inactive flavor is not included in the response.
3. **Given** a flavor's price was recently updated, **When** a customer asks about prices, **Then** the response reflects the updated price immediately.

---

### User Story 3 - Multi-Message Conversation Handling (Priority: P3)

As a customer who writes multiple short messages in quick succession (a common WhatsApp behavior), I want the system to group my messages together and understand my complete intent rather than responding to each fragment individually.

**Why this priority**: WhatsApp users typically break their thoughts into multiple short messages ("Hi", "I want to order", "3 combos of Oreo"). Without message grouping, the AI would waste resources and confuse the user by responding to incomplete fragments.

**Independent Test**: Send 3 to 4 rapid messages within 4 seconds simulating a fragmented order. Verify the system waits for a configurable pause, concatenates all messages, and processes them as a single coherent request.

**Acceptance Scenarios**:

1. **Given** a customer sends "Hola" then "quiero pedir" then "2 combos de avellana" then "para retirar" within 4 seconds, **When** a 4-second pause occurs after the last message, **Then** the system processes all four messages together as one combined request and replies once.
2. **Given** the debounce interval is configurable, **When** the environment setting for the debounce window is changed, **Then** the system uses the configured interval for message grouping.

---

### User Story 4 - Conversational Context Memory (Priority: P4)

As a customer engaged in an ongoing WhatsApp conversation, I want the AI to remember what I said earlier in the same session so I can have a natural back-and-forth (e.g., ask about the menu first, then say "I'll take 2 of the pistachio one").

**Why this priority**: Without conversation memory, each message is treated independently, breaking the natural conversational flow. Customers would need to repeat context, making the experience frustrating.

**Independent Test**: Send a first message asking about the menu, then follow up with a reference to a previously mentioned item. Verify the AI correctly identifies the referenced item.

**Acceptance Scenarios**:

1. **Given** a customer first asks "What do you have?" and receives the menu, **When** they reply "Give me 1 of the pistachio", **Then** the AI remembers the prior exchange and correctly identifies "Pistacho premium" as the requested flavor.
2. **Given** a customer has been inactive for over 2 hours, **When** they send a new message, **Then** the conversation starts fresh without prior context.

---

### User Story 5 - Graceful Error Handling and Guardrails (Priority: P5)

As the business owner, I want the AI to never invent prices or products, to ask clarifying questions when the order is ambiguous (e.g., individual vs. combo not specified), and to handle edge cases like no open cash register or invalid flavor names gracefully.

**Why this priority**: Incorrect pricing or phantom products would cause financial discrepancies and erode customer trust. The AI must be constrained to only use real catalog data.

**Independent Test**: Send an ambiguous order, an order for a non-existent product, and an order when no cash register is open. Verify the AI responds appropriately in each case without fabricating data.

**Acceptance Scenarios**:

1. **Given** a customer says "I want 2 rolls" without specifying a flavor or format, **When** the AI processes the message, **Then** it asks follow-up questions about which flavor and whether they want individual or combo.
2. **Given** no cash register is currently open, **When** a customer tries to place an order, **Then** the system informs the customer that orders cannot be taken at the moment and suggests trying again later.
3. **Given** a customer asks for a flavor that doesn't exist (e.g., "Mango roll"), **Then** the AI politely explains the flavor is not available and presents the current catalog.
4. **Given** the AI has access to tool functions, **Then** it never generates prices from its own knowledge — it always retrieves them from the catalog query tool.

---

### User Story 6 - Webhook Security and Idempotency (Priority: P6)

As a system operator, I want the incoming webhook endpoint to verify its identity with the messaging platform during setup, validate message authenticity via signature verification, and never process the same incoming message twice even if the platform retries delivery.

**Why this priority**: Security and reliability are non-negotiable for a production messaging integration. Duplicate message processing would create duplicate orders, directly impacting revenue accuracy.

**Independent Test**: Send the same webhook payload twice with the same message ID. Verify only one order is created. Send a payload with an invalid signature and verify it is rejected.

**Acceptance Scenarios**:

1. **Given** the messaging platform sends a verification challenge during webhook setup, **When** the system receives the challenge with the correct verification token, **Then** it responds with the challenge value and HTTP 200.
2. **Given** a webhook payload arrives with an invalid cryptographic signature, **When** the system validates the signature, **Then** it rejects the request.
3. **Given** the same message is delivered twice (platform retry), **When** the system checks the message identifier, **Then** it processes the message only once and silently ignores the duplicate.
4. **Given** a webhook payload contains only status update events (sent, delivered, read), **When** the system receives it, **Then** it acknowledges with HTTP 200 but performs no message processing.

---

### User Story 7 - Gestión de Múltiples Direcciones de Envío por Cliente (Priority: P2)

**Como** cliente que pide con envío a domicilio por WhatsApp,  
**quiero** poder registrar una dirección de entrega (calle, altura, piso/depto, referencias, alias) o elegir entre mis direcciones guardadas de pedidos anteriores,  
**para** no tener que escribir mi dirección completa de cero en cada compra y agilizar mis pedidos recurrentes.

- **Independent Test**: Pedir al bot listar las direcciones guardadas asociadas al `wa_id`. Registrar una nueva dirección con alias "Casa" y verificar que se persiste en `CustomerAddress` vinculada al teléfono de la sesión.
- **Acceptance Scenarios**:
  1. **Given** un cliente que solicita envío a domicilio y no tiene direcciones registradas, **When** el agente consulta sus direcciones, **Then** detecta que no tiene ninguna guardada y le solicita calle, altura, piso/depto y referencias para registrarla.
  2. **Given** un cliente con direcciones guardadas (ej. "Casa" y "Trabajo"), **When** solicita envío a domicilio, **Then** el agente lista sus opciones y le pide elegir cuál utilizar.
  3. **Given** un cliente que proporciona datos de una nueva dirección, **When** el agente invoca la herramienta de guardado, **Then** el sistema almacena la dirección vinculándola de forma segura y automática al `wa_id` del cliente.

---

### User Story 8 - Cotización de Envío a Domicilio Previa al Cierre (Priority: P2)

**Como** cliente que elige envío a domicilio,  
**quiero** que el bot me informe el costo de envío calculado hacia mi dirección seleccionada antes de confirmar mi pedido final,  
**para** tener total claridad del desglose económico y decidir si confirmo el delivery o prefiero retiro en tienda.

- **Independent Test**: Elegir productos y seleccionar una dirección guardada. Verificar que el bot cotiza el envío a través del puerto de logística (`DeliveryProvider`) y muestra el costo de envío y total desglosado antes de solicitar confirmación final.
- **Acceptance Scenarios**:
  1. **Given** un cliente con productos seleccionados y una dirección de entrega válida, **When** se evalúa el envío, **Then** el sistema invoca `DeliveryProvider.QuoteDelivery` y presenta el costo estimado de envío al cliente.
  2. **Given** que la dirección está fuera de cobertura o no cotizable, **When** se solicita la cotización, **Then** el sistema le informa que no es posible entregar en esa zona y le ofrece retirar en el local o ingresar otra dirección.

---

### User Story 9 - Generación de Link de Pago y Verificación de Estado (Priority: P2)

**Como** cliente que confirma su pedido,  
**quiero** recibir un link de pago digital (Mercado Pago) junto con mi número de pedido y el importe total desglosado, y poder consultar o confirmar cuando ya realicé el pago,  
**para** que mi pedido pase inmediatamente al estado de preparación y despacho sin demoras.

- **Independent Test**: Confirmar un pedido en WhatsApp. Verificar que se genera la orden en base de datos con estado `PENDING_PAYMENT`, se obtiene la preferencia de pago de `PaymentProvider.GeneratePaymentLink` y se entrega el link al cliente. Luego, enviar "ya pagué" o recibir un webhook de pago aprobado; verificar que la orden transiciona a `PAID`.
- **Acceptance Scenarios**:
  1. **Given** un pedido confirmado por el cliente, **When** el backend crea la orden, **Then** su estado inicial es `PENDING_PAYMENT`, genera el link de pago y se lo envía al cliente.
  2. **Given** un pedido en estado `PENDING_PAYMENT`, **When** el cliente envía "ya pagué" o similar, **Then** el agente consulta el estado con `PaymentProvider.VerifyPaymentStatus` y si está aprobado actualiza la orden a `PAID` y confirma al cliente: "¡Pago acreditado! Tu pedido ya está en preparación".
  3. **Given** un pago aún pendiente o rechazado, **When** se consulta su estado, **Then** el sistema mantiene el pedido en `PENDING_PAYMENT` y le notifica que el pago aún no impactó.
  4. **Given** una notificación entrante por webhook de Mercado Pago (`POST /webhook/mercadopago`), **When** se procesa el evento, **Then** la orden asociada se actualiza a `PAID`.

---

## Edge Cases

- **Mensajes fragmentados rápidos (Debounce de 4 segundos)**: Cliente envía múltiples fragmentos sucesivos en menos de 4 segundos. El buffer y timer consolidan el texto antes de invocar a la IA.
- **Duplicados y notificaciones de estado de Meta**: Mensajes repetidos se ignoran por deduplicación de ID. Notificaciones `statuses` (`sent`, `delivered`, `read`) retornan `HTTP 200` y se descartan inmediatamente.
- **Cliente pide delivery sin direcciones previas**: El bot detecta ausencia de direcciones y solicita los datos para registrarla antes de continuar con la cotización.
- **Caja cerrada al confirmar**: Si no hay caja abierta en `CashRegisterRepository.GetActive()`, el sistema no bloquea abruptamente; le responde al cliente con tono empático informando los horarios o que la caja está cerrada temporalmente.
- **Arqueo de caja y pedidos impagos**: Los pedidos en `PENDING_PAYMENT` no deben distorsionar el arqueo de efectivo o ventas cerradas de la caja activa hasta que su estado sea efectivamente `PAID` o `COMPLETED`.
- **Sesión inactiva por más de 2 horas**: El historial conversacional en memoria se limpia para evitar arrastrar contexto obsoleto.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST accept incoming customer messages from the WhatsApp messaging platform via a webhook endpoint (`POST /webhook/whatsapp`).
- **FR-002**: The system MUST verify webhook registration challenges using a configured verification token (`GET /webhook/whatsapp`).
- **FR-003**: The system MUST validate the authenticity of incoming webhook payloads using cryptographic signature verification when a signing secret is configured.
- **FR-004**: The system MUST deduplicate incoming messages by their unique platform-assigned message identifier to guarantee at-most-once processing.
- **FR-005**: The system MUST acknowledge incoming webhook requests immediately (within the platform's timeout window, < 2s) with `HTTP 200` and process message content asynchronously.
- **FR-006**: The system MUST group multiple rapid-fire messages from the same customer within a configurable time window (default: 4 seconds) and process them as a single combined request.
- **FR-007**: The system MUST use an AI conversational agent to interpret natural language customer messages and determine their intent (catalog inquiry, address management, order placement, or payment verification).
- **FR-008**: The system MUST provide the AI agent with tool functions to query the live product catalog, retrieving only active products and their current prices from the database.
- **FR-009**: The system MUST provide the AI agent with an order creation tool that supports multiple line items per order (e.g., combos and singles of different flavors in the same purchase), passing the customer's phone number strictly from the verified session context (`wa_id`, never from AI parameters). The tool MUST only be invoked after the customer has explicitly confirmed the order summary and delivery modality.
- **FR-010**: The system MUST automatically create a new customer record when a first-time customer places an order, using their WhatsApp phone number (`wa_id`) and profile name.
- **FR-011**: The system MUST look up existing customer records by WhatsApp phone number for returning customers.
- **FR-012**: The system MUST invoke the existing order creation business logic (including cash register validation, price calculation, and customer statistics update) — it MUST NOT duplicate or bypass this logic.
- **FR-013**: The system MUST send reply messages back to the customer through the WhatsApp messaging platform's outbound API.
- **FR-014**: The system MUST maintain per-customer conversation history in memory to enable multi-turn dialogue, with automatic expiration after 2 hours of inactivity.
- **FR-015**: The AI agent MUST never fabricate product names, prices, or availability — it MUST always retrieve this data from the catalog tool.
- **FR-016**: The AI agent MUST ask clarifying questions when the customer's order is ambiguous (e.g., missing flavor selection, unspecified individual vs. combo format, or unclear quantity).
- **FR-017**: The system MUST filter and ignore platform status events (sent, delivered, read) and only process actual customer messages.
- **FR-018**: The system MUST handle errors from the order creation process gracefully and communicate failures to the customer in a user-friendly manner (e.g., "No open register", "Flavor not found").
- **FR-019**: The system MUST support multiple delivery addresses (`1:N`) per customer associated with their WhatsApp phone number (`wa_id`).
- **FR-020**: The system MUST provide the AI agent with tools to list saved customer addresses (`obtener_direcciones_cliente`) and save a new address (`guardar_direccion_cliente`) with street, number, apartment notes, and alias.
- **FR-021**: The system MUST provide a logistics port (`DeliveryProvider`) to quote delivery costs for a given address (`QuoteDelivery`) and request dispatch (`CreateDelivery`), implemented initially with a configurable Mock adapter and prepared for Uber Direct API integration.
- **FR-022**: The system MUST provide a payment gateway port (`PaymentProvider`) to generate a payment link for the order total (products + delivery cost) (`GeneratePaymentLink`) and verify payment status (`VerifyPaymentStatus`), implemented initially with a configurable Mock adapter and prepared for Mercado Pago API integration.
- **FR-023**: The system MUST manage the order lifecycle statuses (`PENDING_PAYMENT`, `PAID`, `PREPARING`, `DISPATCHED`, `DELIVERED`, `CANCELLED`) ensuring that cash register totals and reconciliations are not distorted by unpaid or abandoned orders.
- **FR-024**: The AI agent MUST NEVER use the technical or internal word "topping" when communicating with the customer; it MUST always refer to items as "sabores", "roles", or "variedades".
- **FR-025**: The customer's phone number MUST NEVER be requested from or typed by the user in chat; it MUST be strictly extracted from the WhatsApp session identifier (`wa_id`) and persisted by the backend upon customer creation or lookup.

---

### Key Entities

- **Customer**: Identified by WhatsApp phone number (`wa_id`, unique). Has name, order history count, and timestamps. Pre-existing entity with WhatsApp field already indexed.
- **CustomerAddress**: Linked to `Customer` in a 1:N relationship by `customer_phone` (`wa_id`). Contains `id`, `customer_phone`, `alias` (e.g., "Casa", "Trabajo"), `street_and_number`, `apartment_notes`, `city`, `lat`, `lng`, `is_default`, and timestamps.
- **Order**: Linked to a customer and a cash register. Contains line items, each referencing a flavor with quantity, individual/combo flag, unit price, and subtotal. Extended with:
  - `status`: `OrderStatus` enum (`PENDING_PAYMENT`, `PAID`, `PREPARING`, `DISPATCHED`, `DELIVERED`, `COMPLETED`, `CANCELLED`).
  - `delivery_type`: `DeliveryType` enum (`PICKUP`, `DELIVERY`).
  - `address_id`: Optional foreign key to `CustomerAddress`.
  - `delivery_cost`: Cost of delivery quoted by the logistics provider.
  - `total_amount`: Total order amount (items subtotal + `delivery_cost`).
  - `payment_link`: URL to complete digital payment (Mercado Pago).
  - `external_payment_id`: Payment preference ID from Mercado Pago.
  - `external_delivery_id`: Delivery identifier from Uber Direct.
  - `tracking_url`: Live tracking link from Uber Direct.
- **Topping (Flavor/Product)**: Has name, individual price, combo price, and active/inactive status. Pre-existing catalog entity.
- **Conversation Session**: Ephemeral in-memory entity scoped to a WhatsApp phone number (`wa_id`). Contains the AI conversation history and a message accumulation buffer. Expires after 2 hours of inactivity.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Customers can browse flavors, quote delivery, and receive a payment link via WhatsApp in under 60 seconds from first message to order generation.
- **SC-002**: 100% of catalog inquiries return accurate, real-time flavor names and prices matching the current database state — zero fabricated data.
- **SC-003**: 100% of orders placed via WhatsApp are persisted using the existing order creation logic, producing identical database records and financial calculations.
- **SC-004**: Duplicate messages (same message ID delivered multiple times) result in exactly one order creation — zero duplicate orders.
- **SC-005**: Multi-message sequences sent within the debounce window are consolidated and processed as a single coherent request at least 95% of the time.
- **SC-006**: The system handles at least 50 concurrent customer conversations without degradation, message loss, or cross-conversation data leaks.
- **SC-007**: The webhook responds to the messaging platform within 2 seconds for every request, preventing platform-initiated retries due to timeouts.
- **SC-008**: 100% of saved addresses are bound to the verified session `wa_id` without requiring user phone input.
- **SC-009**: The AI agent never outputs the word "topping" in messages directed to customers.

---

## Assumptions

- The business has a WhatsApp Business API account configured with a phone number and the necessary platform credentials.
- The application is accessible from the internet (or behind a tunnel like ngrok during development) to receive webhook payloads.
- The existing order creation business logic (`OrderUseCase.CreateOrder`) is the single source of truth for order processing, price calculation, and customer statistics.
- The generative AI service uses `google.golang.org/genai` with model `gemini-2.5-flash`, configured via environment variable `GEMINI_API_KEY`.
- Conversation session data (chat history, message buffers) is stored in application memory and is acceptable to lose on restart — persistence of session data is out of scope for v1.
- The AI agent communicates in Spanish (rioplatense dialect) to match the business's customer base.
- Only text messages are processed; media messages (images, voice, documents) are acknowledged with a helpful fallback message.
