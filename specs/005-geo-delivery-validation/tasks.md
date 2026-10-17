# Tasks: Geographic Address Validation & Home Delivery Logistics

**Input**: Design documents from `/specs/005-geo-delivery-validation/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md` (available); no `data-model.md`, `contracts/`, or `quickstart.md` are present.

**Tests**: Includes focused test tasks because each user story defines independent verification scenarios and acceptance cases.

**Organization**: Tasks are grouped by user story in priority order. Paths follow the existing Go Clean Architecture layout (`core/domain`, `core/usecase`, `ports`, `adapters`, `tests`).

## Phase 1: Setup

**Purpose**: Confirm the existing Go service structure and prepare feature packages.

- [X] T001 [P] Map delivery and address feature changes to existing packages in `specs/005-geo-delivery-validation/plan.md`
- [X] T002 [P] Define geocoding and delivery provider configuration keys and mock selection in `README.md`

## Phase 2: Foundational

**Purpose**: Establish shared domain and persistence capabilities required across address, quote, dispatch, and webhook stories.

- [X] T003 Add geocoded result, delivery quote, delivery request, and dispatch result value types in `core/domain/delivery.go`
- [X] T004 Extend delivery provider port with quote, dispatch, cancellation, and quote-expiration error contracts in `ports/output/delivery_provider.go`
- [X] T005 Add order persistence operations for delivery quote and dispatch lifecycle updates in `ports/output/order_repository.go` and `adapters/output/repository/order_repository.go`
- [X] T006 Wire delivery and geocoding provider construction, configuration, and mock selection in `app/app.go`

**Checkpoint**: Shared domain, persistence, and dependency wiring are ready for story work.

## Phase 3: User Story 1 - Address Validation Within Córdoba Capital (Priority: P1) 🎯 MVP

**Goal**: Validate exact Córdoba Capital street addresses before saving, preserve delivery notes, and store normalized address components and coordinates.

**Independent Test**: Submit valid Córdoba Capital addresses, neighboring-town addresses, addresses without street numbers, and unresolved addresses through the WhatsApp address flow; only valid exact addresses are saved with normalized fields and coordinates, and an existing default remains unchanged when a new address is saved.

### Tests for User Story 1

- [X] T007 [P] [US1] Add geocoding provider contract tests for exactness, locality, street number, and coordinate mapping in `adapters/output/geocoding/google_maps_provider_test.go`
- [X] T008 [P] [US1] Add address use case tests for valid, neighboring-town, missing-number, unresolved, notes, and default-address behavior in `core/usecase/address_usecase_test.go`

### Implementation for User Story 1

- [X] T009 [P] [US1] Extend `CustomerAddress` with `RawInput`, `FormattedAddress`, `Neighborhood`, and `ZipCode` fields and Córdoba city default in `core/domain/customer_address.go`
- [X] T010 [P] [US1] Define the geocoding output port and exact-address result contract in `ports/output/geocoding_provider.go`
- [X] T011 [P] [US1] Implement Google Maps Geocoding API adapter with Spanish language, Córdoba component bias, and component parsing in `adapters/output/geocoding/google_maps_provider.go`
- [X] T012 [P] [US1] Implement configurable mock geocoding provider for local use in `adapters/output/geocoding/mock_geocoding_provider.go`
- [X] T013 [US1] Validate geocoding exactness, street number, and Córdoba Capital locality before persistence in `core/usecase/address_usecase.go`
- [X] T014 [US1] Persist normalized address fields and apartment/delivery notes while preserving the current default address in `adapters/output/repository/customer_address_repository.go`
- [X] T015 [US1] Update WhatsApp `save_customer_address` tool schema and result messaging for address validation outcomes in `core/usecase/whatsapp_agent_usecase.go`

**Checkpoint**: Address validation and persistence can be exercised through the WhatsApp agent without relying on delivery quoting.

## Phase 4: User Story 2 - Delivery Quote with Coordinate-Based Coverage Validation (Priority: P1)

**Goal**: Quote delivery from validated coordinates, enforce courier coverage, return ARS pricing, and attach address and fee to the active draft order.

**Independent Test**: Quote saved addresses at near, mid, and outer coverage distances; verify coordinate forwarding, coverage rejection/pickup guidance, fee and duration in ARS, and the draft order total updated with the selected address and delivery fee.

### Tests for User Story 2

- [X] T016 [P] [US2] Add delivery quote use case tests for coordinate input, centavo conversion, coverage rejection, and draft total updates in `core/usecase/delivery_usecase_test.go`
- [X] T017 [P] [US2] Add Uber Direct quote request/response mapping tests in `adapters/output/delivery/uber_direct_provider_test.go`

### Implementation for User Story 2

- [X] T018 [US2] Implement Uber Direct OAuth client-credentials token caching and quote API mapping with coordinates and structured address in `adapters/output/delivery/uber_direct_provider.go`
- [X] T019 [US2] Implement delivery quote orchestration, maximum-radius validation, ARS fee conversion, and draft order update in `core/usecase/delivery_usecase.go`
- [X] T020 [US2] Implement mock delivery quote behavior and configurable local coverage outcomes in `adapters/output/delivery/mock_delivery_provider.go`
- [X] T021 [US2] Add `calculate_delivery_cost` tool schema and invoke quote use case using a saved address in `core/usecase/whatsapp_agent_usecase.go`
- [X] T022 [US2] Include delivery as a separate ARS line item in the WhatsApp order summary and payable total in `core/usecase/order_usecase.go`

**Checkpoint**: A customer can independently validate an address, request a coordinate-based quote, and see the updated draft total.

## Phase 5: User Story 3 - Store Operator Courier Dispatch via App and Live Tracking Notification (Priority: P2)

**Goal**: Allow paid, address-validated orders to be manually dispatched by store operators, persist tracking data, and notify the customer.

**Independent Test**: Dispatch a paid delivery order through `POST /api/orders/{id}/dispatch-delivery`; verify customer and E.164 phone details, delivery notes and small-package data sent to the provider, persisted dispatch ID/tracking URL and `DISPATCHED` status, and WhatsApp tracking notification. Unpaid or unaddressed orders and LLM dispatch attempts must be rejected; expired quotes refresh and retry once.

### Tests for User Story 3

- [X] T023 [P] [US3] Add delivery dispatch use case tests for paid/address gates, E.164 phone, manifest data, persistence, and quote refresh retry in `core/usecase/delivery_usecase_test.go`
- [X] T024 [P] [US3] Add dispatch handler and route tests for `POST /api/orders/{id}/dispatch-delivery` and unpaid-order rejection in `adapters/input/http/handlers/order_handler_test.go`

### Implementation for User Story 3

- [X] T025 [US3] Implement Uber Direct delivery creation, quote-expiration error mapping, manifest, and tracking response parsing in `adapters/output/delivery/uber_direct_provider.go`
- [X] T026 [US3] Implement operator dispatch orchestration, PAID and validated-address checks, E.164 normalization, quote renewal, one retry, and order status updates in `core/usecase/delivery_usecase.go`
- [X] T027 [US3] Add authenticated administrative dispatch endpoint and request/response handling in `adapters/input/http/handlers/order_handler.go`
- [X] T028 [US3] Register `POST /api/orders/{id}/dispatch-delivery` behind management authorization in `adapters/input/http/routes/url_mappings.go`
- [X] T029 [US3] Send the official tracking notification and reheating tips after successful dispatch through `adapters/output/whatsapp/whatsapp_client.go`
- [X] T030 [US3] Ensure the agent exposes only `save_customer_address` and `calculate_delivery_cost`, with no courier dispatch tool, in `core/usecase/whatsapp_agent_usecase.go`
- [X] T031 [US3] Keep payment-triggered auto-dispatch disabled by default and honor `AUTO_DISPATCH_ON_PAYMENT=false` in `app/app.go`

**Checkpoint**: Operators can manually dispatch paid delivery orders and customers receive the tracking link.

## Phase 6: User Story 4 - Courier Status Webhook Updates (Priority: P3)

**Goal**: Verify Uber webhook signatures and update the matching order to delivered or cancelled from trusted status events.

**Independent Test**: Submit validly signed delivery status events and verify the matching order reaches `DELIVERED` or `CANCELLED`; invalid signatures and malformed/unrecognized events do not change order state.

### Tests for User Story 4

- [X] T032 [P] [US4] Add HMAC-SHA256 signature verification and event mapping tests in `adapters/input/http/handlers/uber_delivery_webhook_handler_test.go`
- [X] T033 [P] [US4] Add webhook order status transition tests in `core/usecase/delivery_usecase_test.go`

### Implementation for User Story 4

- [X] T034 [US4] Implement raw-body HMAC-SHA256 verification with constant-time comparison and status event parsing in `adapters/input/http/handlers/uber_delivery_webhook_handler.go`
- [X] T035 [US4] Map verified external delivery IDs to orders and persist DELIVERED/CANCELLED transitions in `core/usecase/delivery_usecase.go`
- [X] T036 [US4] Register the public Uber delivery webhook route and handler in `adapters/input/http/routes/webhook_routes.go`

**Checkpoint**: Only verified courier events can update delivery completion or cancellation state.

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Close cross-story configuration, operational documentation, and integration gaps.

- [X] T037 [P] Document Google Maps, Uber Direct, webhook secret, mock provider, and auto-dispatch environment configuration in `README.md`
- [X] T038 Review provider errors and structured logs across address, quote, dispatch, and webhook flows in `core/usecase/delivery_usecase.go` and `core/usecase/address_usecase.go`
- [X] T039 Verify full address-to-quote-to-operator-dispatch-to-webhook flow against acceptance scenarios in `tests/e2e_test.go`

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; documents existing structure and configuration.
- **Foundational (Phase 2)**: Depends on setup; shared domain/port/persistence/wiring blocks stories.
- **User Stories (Phase 3+)**: Depend on Phase 2. US1 and US2 are both P1 and can proceed in parallel after foundation; US2 requires a validated address at runtime. US3 depends on quote and address capabilities from US1/US2. US4 depends on persisted external delivery IDs from US3.
- **Polish (Phase 7)**: Depends on all stories selected for delivery.

### User Story Dependencies

- **US1 (P1)**: Starts after foundation; no dependency on another story. Suggested MVP.
- **US2 (P1)**: Starts after foundation; integrates with US1's validated address data, so end-to-end quoting follows US1.
- **US3 (P2)**: Requires US1 saved address and US2 quote/order linkage.
- **US4 (P3)**: Requires US3 dispatch IDs persisted on orders.

### Parallel Opportunities

- T001 and T002 can run in parallel; foundational domain and provider port tasks can be split by file where interfaces are settled.
- Within US1, T007-T012 work on distinct test/model/port/adapter files and can proceed concurrently; use case and persistence integration follow their contracts.
- Within US2, provider tests and use case tests can proceed in parallel; provider quote API and orchestration touch separate files.
- Within US3, handler tests and use case tests can be prepared in parallel; handler, route, provider, and notification tasks are separate files after contracts are fixed.
- Within US4, handler signature tests and use case transition tests can proceed in parallel.
- US1 and US2 have parallel implementation opportunities after foundation, though combined end-to-end acceptance requires US1 first.

## Parallel Example: User Story 1

```text
Task: T007 Geocoding provider contract tests in adapters/output/geocoding/google_maps_provider_test.go
Task: T008 Address use case tests in core/usecase/address_usecase_test.go
Task: T009 CustomerAddress fields in core/domain/customer_address.go
Task: T010 Geocoding port in ports/output/geocoding_provider.go
Task: T011 Google Maps adapter in adapters/output/geocoding/google_maps_provider.go
```

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Setup and Foundational phases.
2. Complete US1 address validation and persistence.
3. Verify the US1 independent criteria; deliver validated Córdoba address capture as the first usable increment.

### Incremental Delivery

1. Add US2 coordinate-based quotes and draft totals.
2. Add US3 operator-controlled dispatch and tracking notification.
3. Add US4 signed courier status webhooks.
4. Complete cross-story configuration and integration review.

## Notes

- `[P]` marks tasks that can be performed independently in different files after their stated contracts are available.
- Story tasks carry `[US#]` labels; setup, foundational, and polish tasks do not.
- Test task descriptions are included because the specification defines independent test scenarios for every story.
