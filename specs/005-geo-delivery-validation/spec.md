# Feature Specification: Geographic Address Validation & Home Delivery Logistics

**Feature Branch**: `005-geo-delivery-validation`

**Created**: 2026-10-16

**Status**: Draft

**Input**: User description: "Design and implement the complete Geographic Address Validation (via Google Maps Geocoding API) and Home Delivery Logistics (via Uber Direct API) subsystem for Córdoba Capital, integrated into the WhatsApp AI Agent flow and strictly following Clean Architecture and Hexagonal Architecture (Ports & Adapters) in Go."

## Clarifications

### Session 2026-10-17

- Q: ¿Cómo debe exponer el backend el despacho de Uber para la app y cómo debe manejarse la activación del envío tras el pago? → A: Endpoint REST dedicado para la app (ej. `POST /api/orders/{id}/dispatch-delivery`) con control manual del operador de la tienda y auto-despacho automático desactivado por defecto (`AUTO_DISPATCH_ON_PAYMENT=false`).
- Q: ¿El tool de cálculo de costo de envío del LLM debe persistir automáticamente la dirección y el costo cotizado en el pedido en borrador (draft) para actualizar el total a pagar? → A: Sí, el tool del LLM debe actualizar automáticamente el pedido en borrador con la dirección y el costo de envío, reflejándolo en el total a pagar.
- Q: ¿Qué capacidades de tool calling debe tener el agente LLM respecto a la logística de envíos? → A: El agente LLM SOLO tiene tools para validar/guardar direcciones (`save_customer_address`) y calcular/cotizar el costo de envío (`calculate_delivery_cost`). El agente LLM NO tiene ni tendrá ningún tool para solicitar o despachar pedidos a Uber (`request_courier`/`dispatch_delivery` queda estrictamente fuera de su alcance).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Address Validation Within Córdoba Capital (Priority: P1)

A customer chatting with the Canerollss WhatsApp AI agent provides a delivery address (e.g., "Av. Colón 1500, 4to B"). The system validates that the address is a real, exact street address within Córdoba Capital, normalizes it with its official formatted form, captures the neighborhood, zip code, and precise geographic coordinates, and stores it along with any apartment/floor notes. If the address is outside Córdoba Capital, lacks a street number, or cannot be found, the agent politely asks the customer to verify and correct their address.

**Why this priority**: Without validated addresses, no delivery can be dispatched reliably. Address validation is the foundation of the entire delivery subsystem — all downstream operations (quoting, dispatching, tracking) depend on having a verified address with exact coordinates.

**Independent Test**: Can be fully tested by sending various address inputs through the WhatsApp agent (valid Córdoba addresses, addresses in neighboring towns, streets without numbers, non-existent streets) and verifying that only valid Córdoba Capital addresses with street numbers are accepted and stored with their normalized data and coordinates.

**Acceptance Scenarios**:

1. **Given** a customer in WhatsApp sends a valid Córdoba Capital address with street number (e.g., "Av. Colón 1500, 4to B"), **When** the agent processes the address save request, **Then** the system validates the address, confirms it belongs to Córdoba Capital, stores the formatted address with coordinates (`lat`/`lng`), neighborhood, zip code, and apartment notes ("4to B"), and confirms the address was saved successfully.
2. **Given** a customer sends an address without a street number (e.g., "Av. Colón, Córdoba"), **When** the system validates it, **Then** the address is rejected with a clear message asking the customer to include the street number (altura).
3. **Given** a customer sends an address that belongs to a neighboring town (e.g., "Av. San Martín 200, Villa Allende"), **When** the system validates it, **Then** the address is rejected with a message explaining that deliveries are only available within Córdoba Capital.
4. **Given** a customer sends a non-existent or unrecognizable address (e.g., "Calle Inventada 9999"), **When** the system validates it, **Then** the address is rejected with a message asking the customer to verify the street name and number.
5. **Given** a customer already has saved addresses and saves a new one, **When** the new address is validated and saved, **Then** the first address remains as default unless the customer explicitly changes it.

---

### User Story 2 - Delivery Quote with Coordinate-Based Coverage Validation (Priority: P1)

After a customer selects a validated delivery address for their order, the WhatsApp AI agent invokes a tool to obtain a real-time delivery cost quote using the stored coordinates. This ensures the delivery pin lands at the exact door location, validates that the distance falls within the operational coverage radius of the motorcycle courier service from the Canerollss store in Villa Cabrera, persists the delivery address and quoted fee in the customer's draft order to update the total payable amount, and presents the actual cost in Argentine pesos (ARS) to the customer before they confirm the order.

**Why this priority**: The delivery quote directly affects the customer's purchase decision and the financial accuracy of every delivery order. Without coordinate-based quoting, deliveries could be dispatched to incorrect locations or outside the coverage area, causing failed deliveries and financial losses.

**Independent Test**: Can be fully tested by requesting quotes for validated addresses at varying distances from the store (nearby in Villa Cabrera, mid-range in Nueva Córdoba, far edge in a distant Córdoba barrio) and verifying that accurate costs in ARS are returned for reachable addresses, while out-of-range addresses receive a clear coverage error, and that draft orders correctly reflect the delivery line item in their total.

**Acceptance Scenarios**:

1. **Given** a validated customer address with coordinates within the courier coverage radius, **When** the system requests a delivery quote, **Then** it returns the estimated delivery cost in ARS, estimated delivery duration in minutes, a quote identifier, and a quote expiration time.
2. **Given** a validated customer address whose coordinates place it outside the maximum courier range from the store, **When** the system requests a delivery quote, **Then** it informs the customer that the delivery zone exceeds the coverage area and offers the option of picking up the order at the store.
3. **Given** the delivery cost is presented to the customer, **When** the customer views their order summary before confirmation, **Then** the delivery cost is shown as a separate line item in ARS alongside the product total.
4. **Given** a customer with an active order draft who requests delivery, **When** the LLM tool executes the quote calculation, **Then** the system automatically updates the draft order with the selected delivery address ID and the quoted delivery fee, ensuring the calculated payment total includes the delivery fee.

---

### User Story 3 - Store Operator Courier Dispatch via App and Live Tracking Notification (Priority: P2)

Once a delivery order is confirmed and paid (`PAID` status), the store operators can view the order in preparation from their management app and manually trigger the motorcycle courier dispatch via an administrative REST endpoint. The WhatsApp AI agent explicitly has NO tool calling capability to dispatch or request couriers. When dispatch is triggered by the operator, the backend dispatches a motorcycle courier providing the customer's name, WhatsApp phone number in international format, delivery instructions (floor, apartment, references), and small-package dimensions. The system captures the live tracking link, updates the order to dispatched status, and sends the customer an automatic WhatsApp message with the tracking link and product reheating tips.

**Why this priority**: This completes the delivery lifecycle from order confirmation to active delivery under human store operator supervision. While P2 because it depends on P1 stories being functional, it is essential for the core business value of home delivery.

**Independent Test**: Can be fully tested by creating a paid delivery order with a validated address and invoking the dispatch endpoint (`POST /api/orders/{id}/dispatch-delivery`), then verifying the courier is requested with correct customer details (name, E.164 phone, delivery instructions), the tracking link is captured and stored on the order, the order status changes to dispatched, and the customer receives the WhatsApp notification with tracking link. Attempts to dispatch unpaid orders or requests from the LLM agent must be rejected.

**Acceptance Scenarios**:

1. **Given** a delivery order in `PAID` status with a validated address, **When** a store operator triggers dispatch via the management REST endpoint (`POST /api/orders/{id}/dispatch-delivery`), **Then** the system requests a motorcycle courier with the customer's name, phone in `+549...` format, delivery instructions from the address apartment notes, and small package dimensions, and receives back an external delivery ID and tracking link.
2. **Given** a successful courier dispatch, **When** the tracking link is received, **Then** the order is updated to `DISPATCHED` status with the external delivery ID and tracking URL persisted, and the customer receives a WhatsApp message with the official "🚚✨ ¡Canerollss en camino! ✨" template including the live tracking link and microwave reheating tips.
3. **Given** a dispatch attempt where the previously obtained quote has expired (older than 15 minutes), **When** the system detects the expiration, **Then** it automatically and transparently obtains a fresh quote and retries the dispatch without requiring customer intervention.
4. **Given** a dispatch attempt that fails due to the quote expiration error from the courier service, **When** the error is received, **Then** the system automatically obtains a new quote and retries the dispatch once, failing gracefully with a notification if the retry also fails.
5. **Given** an order that is NOT in `PAID` status, **When** dispatch is triggered via the administrative endpoint, **Then** the request is rejected with a 400 Bad Request indicating the order must be paid before dispatching.

---

### User Story 4 - Courier Status Webhook Updates (Priority: P3)

The system receives real-time status updates from the courier service via a secure webhook endpoint. When a delivery is completed or cancelled by the courier service, the corresponding order is automatically updated to reflect the final status.

**Why this priority**: While important for order lifecycle completeness and operational visibility, the core delivery flow works end-to-end without real-time webhook updates. Status can be tracked manually via the tracking link in the interim.

**Independent Test**: Can be fully tested by sending simulated webhook payloads (delivery completed, delivery cancelled) to the webhook endpoint with valid cryptographic signatures, and verifying orders are updated to the correct final status. Invalid signatures should be rejected.

**Acceptance Scenarios**:

1. **Given** an order in `DISPATCHED` status, **When** the system receives a verified webhook notification indicating delivery completion, **Then** the order status is updated to `DELIVERED`.
2. **Given** an order in `DISPATCHED` status, **When** the system receives a verified webhook notification indicating delivery cancellation, **Then** the order status is updated to `CANCELLED`.
3. **Given** a webhook request with an invalid or missing cryptographic signature, **When** the system receives it, **Then** the request is rejected and no order state changes occur.

---

### Edge Cases

- What happens when the geocoding service is temporarily unavailable? The system returns a clear error message to the customer asking them to try again shortly, without saving any incomplete address data.
- What happens when the courier service is temporarily unavailable for quoting? The system informs the customer that delivery cost cannot be calculated at the moment and suggests trying again or choosing store pickup.
- What happens when a customer provides an address at the boundary of Córdoba Capital (e.g., on a border street shared with a neighboring municipality)? The system relies on the geocoding service's locality classification; if classified as Córdoba Capital, it is accepted.
- What happens when multiple quotes are requested for the same address in quick succession? Each request generates a fresh quote; the system uses the most recent valid quote for dispatch.
- What happens when a delivery order is dispatched but the customer's phone number format is non-standard? The system normalizes the WhatsApp ID to E.164 format (`+` followed by digits) before sending to the courier service.
- What happens when the courier service returns a quote in a sub-unit currency (e.g., centavos)? The system converts to the standard unit (dividing by 100) before presenting or storing the amount.
- What happens when dispatch is requested for an unpaid order or an order lacking coordinates? The dispatch endpoint returns an error, preventing courier dispatch.
- What happens if the LLM attempts to call a delivery creation tool? No such tool is provided or defined in the LLM tool declaration, preventing the model from initiating courier requests.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST validate every new customer address against a geocoding service before persisting it, confirming the address is a real, exact street-level location (with street number) within Córdoba Capital.
- **FR-002**: System MUST extract and store the normalized/formatted address, street and number, neighborhood, city, zip code, and precise geographic coordinates (latitude and longitude) from the geocoding result.
- **FR-003**: System MUST reject addresses that lack a street number, are outside Córdoba Capital, or cannot be resolved to a specific street-level location, returning descriptive error messages suitable for the WhatsApp AI agent to relay to the customer.
- **FR-004**: System MUST store apartment/floor/delivery instructions (e.g., "4to B", "Portón negro, timbre 3") as separate notes alongside the validated address.
- **FR-005**: System MUST obtain delivery cost quotes using the validated address's exact coordinates (latitude and longitude) and structured address data, ensuring the courier service's delivery pin matches the precise door location.
- **FR-005b**: The WhatsApp AI Agent MUST be provided with tool calling capabilities strictly limited to address validation (`save_customer_address`) and delivery cost quoting (`calculate_delivery_cost`). When `calculate_delivery_cost` is invoked, the system MUST automatically link the delivery address and quoted delivery fee to the customer's active draft order so the order total reflects the delivery cost.
- **FR-006**: System MUST present the delivery cost to the customer in Argentine pesos (ARS), converting from sub-unit amounts if the courier service returns values in centavos.
- **FR-007**: System MUST detect when a delivery destination exceeds the motorcycle courier's maximum coverage radius from the store and inform the customer, offering store pickup as an alternative.
- **FR-008**: System MUST implement complete backend capabilities for motorcycle courier dispatch via an administrative REST endpoint (e.g., `POST /api/orders/{id}/dispatch-delivery`) intended for the future store management app, passing the customer's name, WhatsApp phone number in international E.164 format, delivery instructions, and small-package dimensions.
- **FR-009**: System MUST capture the live tracking URL returned by the courier service upon successful dispatch and persist it on the order along with the external delivery identifier.
- **FR-010**: System MUST update the order status to `DISPATCHED` upon successful courier dispatch and automatically send the customer a WhatsApp message containing the live tracking URL and product reheating tips.
- **FR-011**: System MUST handle quote expiration (15-minute window) by automatically and transparently obtaining a fresh quote before retrying dispatch, without requiring customer intervention.
- **FR-012**: System MUST expose a secure webhook endpoint that validates incoming requests using cryptographic signature verification (HMAC-SHA256) and updates order status to `DELIVERED` or `CANCELLED` based on verified courier service notifications.
- **FR-013**: System MUST support mock/simulated providers for both geocoding and courier services when configured for local development, allowing the full flow to be tested without external API credentials.
- **FR-014**: System MUST normalize the customer's WhatsApp ID to E.164 international format (`+` prefix followed by country code and number) before sending it to the courier service.
- **FR-015**: System MUST NOT expose any dispatch or courier request tool calling capability to the WhatsApp AI Agent. Automatic immediate dispatch on payment MUST be disabled by default (`AUTO_DISPATCH_ON_PAYMENT=false`), requiring an explicit operator trigger from the management app endpoint.
- **FR-016**: The dispatch endpoint MUST verify that the target order is in `PAID` status and has an associated validated delivery address before requesting the courier from the logistics provider.

### Key Entities *(include if feature involves data)*

- **Customer Address**: Represents a validated, geocoded delivery location for a customer. Contains the raw input from the customer, the official normalized address, street and number, apartment/delivery notes, neighborhood, city, zip code, geographic coordinates, and a default flag. Each customer can have multiple addresses.
- **Geocoded Result**: Represents the output of a geocoding validation. Contains the formatted address, individual address components (street, number, neighborhood, city, state, zip code), geographic coordinates, location precision type, and an exactness flag indicating whether the result is a precise street-level address.
- **Delivery Quote**: Represents a time-limited cost estimate for delivering an order to a specific address. Contains the quote identifier, delivery fee in ARS, currency code, estimated duration in minutes, estimated arrival time, and expiration time.
- **Delivery Request**: Represents all the information needed to dispatch a courier. Contains the order reference, quote identifier, customer name, customer phone in E.164 format, validated destination address, delivery instructions, and a list of items with names, quantities, weights, and size classification.
- **Delivery Dispatch Result**: Represents the outcome of a successful courier dispatch. Contains the external delivery identifier, courier status, live tracking URL, and final fee.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of addresses saved through the system are verified as real, exact street-level locations within Córdoba Capital with valid geographic coordinates — zero unvalidated addresses are persisted.
- **SC-002**: Customers receive clear, actionable feedback within 5 seconds when an address is invalid (wrong city, missing street number, or non-existent), enabling them to correct and resubmit without confusion.
- **SC-003**: Delivery cost quotes reflect the actual courier fee based on precise geographic coordinates, with zero discrepancy between the quoted amount and the amount charged at dispatch.
- **SC-004**: Orders dispatched via the system result in the courier arriving at the correct door location (matching the geocoded coordinates) in 95% or more of deliveries.
- **SC-005**: Customers receive the live tracking link via WhatsApp within 10 seconds of successful courier dispatch.
- **SC-006**: Quote expiration is handled transparently — 100% of dispatch attempts with expired quotes automatically refresh the quote and retry without customer-facing errors.
- **SC-007**: The webhook endpoint correctly processes 100% of validly signed courier status notifications, updating order status within 3 seconds of receipt.
- **SC-008**: The full address validation → quote → dispatch → tracking notification flow completes end-to-end in under 30 seconds from the customer's perspective.
- **SC-009**: 0% of courier dispatch requests are initiated by the LLM agent; 100% of dispatches are authorized through the administrative management endpoint after payment confirmation.

## Assumptions

- The business operates deliveries exclusively within Córdoba Capital, Argentina. Neighboring localities (Villa Allende, Carlos Paz, Río Ceballos, etc.) are explicitly excluded.
- All deliveries use motorcycle couriers for small packages (canelones and similar food products). Car-based delivery is not needed.
- The store has a single, fixed pickup location in Villa Cabrera, Córdoba, with known coordinates. Multi-store support is not in scope.
- Customers interact exclusively through WhatsApp for menu browsing, order placement, address submission, and delivery quotes. Delivery dispatch, however, is controlled by store staff via an administrative backend REST API endpoint (`POST /api/orders/{id}/dispatch-delivery`) intended for a future management app.
- The WhatsApp AI agent is strictly forbidden from triggering courier dispatch. Tool calling for the LLM is limited to address validation (`save_customer_address`) and delivery cost quoting (`calculate_delivery_cost`).
- The geocoding service is restricted to Spanish-language results for consistency with the Argentine address format.
- Quote validity is approximately 15 minutes as per the courier service's standard policy. The system treats any quote older than 15 minutes as expired.
- The courier service's fee is returned in centavos (sub-unit) and must be converted to pesos (dividing by 100) for display and storage.
- Phone numbers from the WhatsApp platform are received as `wa_id` values (digits only, e.g., `5493515551234`) and must be prefixed with `+` to form E.164 format.
- The existing order entity already has fields for external delivery ID, tracking URL, and delivery cost, which will be used by this feature.
- The existing customer address entity will be extended with additional fields (formatted address, neighborhood, zip code) as needed.
- Auto-dispatch on payment is disabled by default (`AUTO_DISPATCH_ON_PAYMENT=false`), ensuring human operators trigger courier dispatch from the management app when orders are prepared and ready for pickup.
- Mock providers for both geocoding and courier services are available for local development and testing, selected via the `USE_MOCK_PROVIDERS` environment variable.
