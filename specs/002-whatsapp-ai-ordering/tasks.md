# Tasks: WhatsApp AI-Powered Order Automation & Transactional Flow

**Branch**: `002-whatsapp-ai-ordering`  
**Input**: [spec.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/002-whatsapp-ai-ordering/spec.md), [plan.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/002-whatsapp-ai-ordering/plan.md), [data-model.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/002-whatsapp-ai-ordering/data-model.md), [contracts/](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/002-whatsapp-ai-ordering/contracts/)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, SDK installation, and configuration management.

- [X] T001 Install Google Gen AI official SDK dependency `google.golang.org/genai` in `go.mod`
- [X] T002 [P] Configure environment variables and fallback defaults (`GEMINI_API_KEY`, `WHATSAPP_VERIFY_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_API_TOKEN`, `USE_MOCK_PROVIDERS`) in `.env` and `adapters/output/datasource/connection.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data models, database migrations, repository ports, and external logistics/payment provider interfaces.

**⚠️ CRITICAL**: Must be completed before any user story can be executed.

- [X] T003 [P] Create `CustomerAddress` domain entity in `core/domain/customer_address.go` with fields `id` (PK), `customer_phone` (varchar 20, index), `alias` (varchar 50, default 'Casa'), `street_and_number` (varchar 255), `apartment_notes` (varchar 255), `city` (varchar 100), `lat`, `lng`, `is_default` (bool), and timestamps
- [X] T004 [P] Extend `Order` domain entity in `core/domain/order.go` with statuses (`PENDING_PAYMENT`, `PAID`, `PREPARING`, `DISPATCHED`, `DELIVERED`, `COMPLETED`, `CANCELLED`), `delivery_type` (`PICKUP`, `DELIVERY`), `address_id` (*uint, FK), `delivery_cost` (float64), `total_amount` (float64), `payment_link` (varchar 500), `external_payment_id` (varchar 100), `external_delivery_id` (varchar 100), and `tracking_url` (varchar 500)
- [X] T005 [P] Create `CustomerAddressRepository` port interface in `ports/output/customer_address_repository.go` (`Save`, `FindByPhone`, `FindByID`, `SetDefault`, `Delete`)
- [X] T006 [P] Implement `CustomerAddressRepository` with GORM in `adapters/output/repository/customer_address_repository.go`
- [X] T007 Register `&domain.CustomerAddress{}` in `AutoMigrate` inside `adapters/output/datasource/connection.go`
- [X] T008 [P] Create `DeliveryProvider` logistics port interface in `ports/output/delivery_provider.go` (`QuoteDelivery`, `CreateDelivery`)
- [X] T009 [P] Create `PaymentProvider` payment port interface in `ports/output/payment_provider.go` (`GeneratePaymentLink`, `VerifyPaymentStatus`)
- [X] T010 [P] Implement `MockDeliveryProvider` in `adapters/output/delivery/mock_delivery_provider.go` with configurable fee, quote generation, and mock tracking URLs
- [X] T011 [P] Implement `UberDirectProvider` HTTP adapter skeleton in `adapters/output/delivery/uber_direct_provider.go` pointing to `/v1/customers/{id}/delivery_quotes` and `/deliveries`
- [X] T012 [P] Implement `MockPaymentProvider` in `adapters/output/payment/mock_payment_provider.go` generating mock checkout links and status verification
- [X] T013 [P] Implement `MercadoPagoProvider` HTTP adapter skeleton in `adapters/output/payment/mercadopago_provider.go` pointing to `/checkout/preferences` and `/v1/payments/{id}`
- [X] T014 [P] Implement `SessionManager` in `adapters/input/http/session/session_manager.go` with `sync.RWMutex`, 4-second `time.Timer` debounce buffer, and multi-turn message history per `wa_id`

**Checkpoint**: Foundation ready — all domain entities, repositories, provider ports, and session debounce buffers are ready for story implementations.

---

## Phase 3: User Story 1 - Customer Builds and Confirms an Order via WhatsApp (Priority: P1) 🎯 MVP

**Goal**: Enable customers to order cinnamon rolls in natural language, choose delivery modality, review itemized totals without premature order insertion, confirm the purchase, and receive an order ID in `PENDING_PAYMENT` with a payment link. The customer's phone number is never typed; it is automatically bound from the WhatsApp session (`wa_id`).

**Independent Test**: Send order request in natural language. Verify the agent displays the itemized price breakdown and asks for delivery modality without creating a database order. Then, send "Confirmo el pedido para retirar"; verify the order is registered with `status = PENDING_PAYMENT`, `delivery_type = PICKUP`, customer profile created/linked by `wa_id`, and a payment link is returned.

- [X] T015 [US1] Implement `WhatsAppAgentUseCase` in `core/usecase/whatsapp_agent_usecase.go` initializing `genai.Client` (`gemini-2.5-flash`), setting rioplatense system instructions (strictly forbidding the word "topping", using "sabores"/"roles"), and managing conversation turns
- [X] T016 [US1] Implement `crear_pedido_con_link_pago` tool declaration and execution runner in `core/usecase/whatsapp_agent_usecase.go`, supporting multiple line items (combos and singles of different flavors), resolving/creating customer by `wa_id`, persisting order in `PENDING_PAYMENT`, and invoking `PaymentProvider.GeneratePaymentLink`
- [X] T017 [US1] Implement outbound WhatsApp Cloud API client helper in `adapters/output/whatsapp/whatsapp_client.go` to send messages via Meta Graph API (or log in dev mode)
- [X] T018 [US1] Implement `WhatsAppHandler` in `adapters/input/http/handlers/whatsapp_handler.go` with `VerifyWebhook` (GET challenge) and `HandleWebhook` (POST with immediate HTTP 200, status filtering, message text extraction, and async dispatch to `SessionManager`)
- [X] T019 [US1] Register `/webhook/whatsapp` routes in `adapters/input/http/routes/webhook_routes.go` and wire all dependencies in `app/app.go`

**Checkpoint**: User Story 1 is functional. The core MVP flow works end-to-end for pickup orders with confirmation and payment links.

---

## Phase 4: User Story 2 - Customer Inquires About the Menu and Prices (Priority: P2)

**Goal**: Customer asks for flavors and prices; the agent queries live database toppings and responds using natural flavor terms without inventing data or using "topping".

**Independent Test**: Send "Qué sabores tienen?", verify live active toppings from the database are returned with individual and combo prices without the word "topping".

- [X] T020 [US2] Implement catalog inquiry function/tool `consultar_catalogo` in `core/usecase/whatsapp_agent_usecase.go` querying `ToppingRepository.GetAllActive()` and formatting active flavors and prices

**Checkpoint**: User Stories 1 and 2 operate coherently.

---

## Phase 5: User Story 7 - Gestión de Múltiples Direcciones de Envío por Cliente (Priority: P2)

**Goal**: 1:N address persistence and selection per customer linked securely to WhatsApp `wa_id`.

**Independent Test**: Inquire about saved addresses -> returns empty or saved list; send address details -> saves address in `customer_address` with alias "Casa" bound to `wa_id`.

- [X] T021 [US7] Implement `AddressUseCase` in `core/usecase/address_usecase.go` with `GetCustomerAddresses`, `SaveAddress`, and `GetAddressByID`
- [X] T022 [US7] Register and wire tools `obtener_direcciones_cliente` and `guardar_direccion_cliente` in `core/usecase/whatsapp_agent_usecase.go` injecting session `wa_id`

**Checkpoint**: Customers can manage multiple saved delivery addresses.

---

## Phase 6: User Story 8 - Cotización de Envío a Domicilio Previa al Cierre (Priority: P2)

**Goal**: Pre-order delivery fee quotation via `DeliveryProvider` towards a selected saved address before confirming the order.

**Independent Test**: Request delivery to a saved address; verify the agent calculates shipping fee via `DeliveryProvider` and shows the total before confirmation.

- [X] T023 [US8] Register and wire tool `cotizar_envio` in `core/usecase/whatsapp_agent_usecase.go` invoking `DeliveryProvider.QuoteDelivery(ctx, address)` and presenting estimated fee and time

**Checkpoint**: Delivery quoting is fully integrated before order confirmation.

---

## Phase 7: User Story 9 - Generación de Link de Pago y Verificación de Estado (Priority: P2)

**Goal**: Receive Mercado Pago payment link upon confirmation, allow customer to say "ya pagué", or receive webhook payment notification transitioning order to `PAID`.

**Independent Test**: Confirm order with delivery -> receive link; send "ya pagué" or POST `/webhook/mercadopago` -> order transitions to `PAID` and customer receives confirmation.

- [X] T024 [US9] Register and wire tool `verificar_estado_pago` in `core/usecase/whatsapp_agent_usecase.go` invoking `PaymentProvider.VerifyPaymentStatus` and updating order status to `PAID`
- [X] T025 [US9] Implement `MercadoPagoHandler` in `adapters/input/http/handlers/mercadopago_handler.go` to process `POST /webhook/mercadopago` payment IPNs, update order to `PAID`, and notify the customer via WhatsApp
- [X] T026 [US9] Register `POST /webhook/mercadopago` in `adapters/input/http/routes/webhook_routes.go` and wire in `app/app.go`

**Checkpoint**: Full transactional loop (Order -> Payment Link -> Verification / Webhook -> Paid Status) complete.

---

## Phase 8: User Story 3 - Multi-Message Conversation Handling (Priority: P3)

**Goal**: Ensure rapid fragmented messages (< 4s) are batched and processed as a single prompt.

**Independent Test**: Send 4 rapid messages in < 4s; verify a single consolidated reply is produced.

- [X] T027 [US3] Verify debounce window timer handling and message concatenation in `adapters/input/http/session/session_manager.go` under concurrent bursts

---

## Phase 9: User Story 4 - Conversational Context Memory (Priority: P4)

**Goal**: Multi-turn conversation context memory per `wa_id` with 2-hour inactivity cleanup.

**Independent Test**: Send menu inquiry, then follow up with "Dame 2 del primero"; verify the agent retains context and correctly resolves the flavor.

- [X] T028 [US4] Manage sliding multi-turn history (`[]*genai.Content`) and 2-hour expiration timer per user session in `adapters/input/http/session/session_manager.go`

---

## Phase 10: User Story 5 & 6 - Guardrails, Webhook Security & Idempotency (Priority: P5/P6)

**Goal**: Webhook idempotency (ignoring duplicate `wamid` messages), graceful handling of closed cash registers, and customer language protection.

- [X] T029 [US6] Implement in-memory message ID deduplication cache in `adapters/input/http/handlers/whatsapp_handler.go`
- [X] T030 [US5] Implement cash register validation guardrail in `WhatsAppAgentUseCase` informing user politely if no register is open without breaking the conversation

---

## Phase 11: Polish & Verification

**Purpose**: Automated tests, clean compilation, and quickstart end-to-end verification.

- [X] T031 [P] Create unit tests for `SessionManager` debouncing in `adapters/input/http/session/session_manager_test.go`
- [X] T032 [P] Create unit tests for `WhatsAppAgentUseCase` tool runner and order creation in `core/usecase/whatsapp_agent_usecase_test.go`
- [X] T033 Execute `go mod tidy` and verify `go build ./...` compiles with 0 errors
- [X] T034 Validate the 5 scenarios described in `specs/002-whatsapp-ai-ordering/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Phase 1 — **BLOCKS all user stories**.
- **User Story 1 (Phase 3 - MVP)**: Depends on Phase 2.
- **User Stories 2, 7, 8, 9 (Phases 4-7)**: Depend on Phase 2 and integrate cleanly with Story 1.
- **User Stories 3, 4, 5, 6 (Phases 8-10)**: Non-blocking enhancements to session and security.
- **Polish & Verification (Phase 11)**: Depends on all implementation tasks.

### Parallel Opportunities

- Within Phase 2: Tasks T003, T004, T005, T008, T009, T010, T011, T012, T013, T014 can be developed in parallel as they operate on distinct files and interfaces.
- Within Phase 11: Unit test tasks T031 and T032 can be executed in parallel.

---

## Implementation Strategy: MVP First

1. **Step 1**: Complete Phase 1 (Setup) and Phase 2 (Foundational data models, repositories, provider ports, session manager).
2. **Step 2**: Implement Phase 3 (User Story 1 - MVP): Pickup order creation with confirmation and payment link.
3. **Step 3**: Incrementally enable Phase 5 (US7: Addresses), Phase 6 (US8: Delivery Quote), Phase 7 (US9: Payment verification and Webhook), and Phase 4 (US2: Catalog).
4. **Step 4**: Run Phase 11 automated verification (`go test ./...` and `go build ./...`).
