# Research: Geographic Address Validation & Home Delivery Logistics

**Feature**: `005-geo-delivery-validation`
**Date**: 2026-10-16

## R-001: Google Maps Geocoding API Contract & Córdoba Filtering

**Decision**: Use Google Maps Geocoding API with `components` parameter for geographic restriction to Córdoba, Argentina, combined with post-response validation of `address_components` to confirm `locality` or `administrative_area_level_2` matches "Córdoba".

**Rationale**:
- The `components=locality:Cordoba|administrative_area:Cordoba|country:AR` filter biases results but does not strictly exclude other localities. Post-response validation is mandatory.
- The result `types` array at the result level must include `street_address`, `premise`, or `subpremise` to confirm an exact street-level match (not a route, locality, or approximate result).
- The `geometry.location_type` field (`ROOFTOP`, `RANGE_INTERPOLATED`, `GEOMETRIC_CENTER`, `APPROXIMATE`) indicates accuracy. `ROOFTOP` is ideal; `RANGE_INTERPOLATED` is acceptable for street addresses.
- `address_components` with type `street_number` must exist to confirm the address has a valid altura/numeración.

**Alternatives Considered**:
- **Google Places API (Autocomplete)**: Provides UI-oriented suggestions but requires a different billing model and doesn't fit the backend-only WhatsApp flow.
- **Nominatim (OpenStreetMap)**: Free but less accurate for Argentine street addresses, especially in Córdoba barrios.

**API Contract Summary**:
- Endpoint: `GET https://maps.googleapis.com/maps/api/geocode/json`
- Query params: `address`, `components`, `language=es`, `key`
- Response: `status` ("OK", "ZERO_RESULTS"), `results[].formatted_address`, `results[].address_components[]`, `results[].geometry.location.{lat,lng}`, `results[].geometry.location_type`, `results[].types[]`

---

## R-002: Uber Direct API Authentication (OAuth 2.0 Client Credentials)

**Decision**: Implement a thread-safe OAuth 2.0 token manager using `sync.RWMutex` that caches the Bearer token in memory and refreshes it before expiration.

**Rationale**:
- Uber Direct uses `POST https://auth.uber.com/oauth/v2/token` with `grant_type=client_credentials` and `scope=eats.deliveries`.
- Tokens expire after `expires_in` seconds (typically 2,592,000 = 30 days), but we should refresh proactively (e.g., when within 5 minutes of expiry).
- Thread-safety via `sync.RWMutex` is essential since multiple goroutines (webhook handlers, agent requests) may concurrently need the token.
- Content-Type must be `application/x-www-form-urlencoded`.

**Alternatives Considered**:
- **Per-request token fetch**: Simpler but wasteful and slower. Rejected.
- **External token store (Redis)**: Overkill for a single-instance deployment. Rejected.

---

## R-003: Uber Direct Delivery Quotes API Contract

**Decision**: Use `POST /v1/customers/{customer_id}/delivery_quotes` with both structured JSON-stringified address fields and explicit latitude/longitude coordinates.

**Rationale**:
- The `dropoff_address` field must be a JSON-stringified object: `{"street_address": ["..."], "city": "Córdoba", "state": "Córdoba", "zip_code": "5000", "country": "AR"}`.
- `dropoff_latitude` and `dropoff_longitude` are provided alongside for precise pin placement.
- The `fee` in the response is in **centavos** (integer). Must divide by 100.0 for ARS display.
- The quote response includes `id` (quote_id), `fee`, `currency`, `duration` (minutes), and an implicit ~15-minute expiry window.

**Alternatives Considered**:
- **Address-only quoting (no coordinates)**: Uber may geocode to incorrect pin. Rejected — coordinates ensure exact door placement.

---

## R-004: Uber Direct Create Delivery API Contract

**Decision**: Use `POST /v1/customers/{customer_id}/deliveries` with `quote_id`, structured pickup/dropoff data, `deliverable_action: "deliverable_action_meet_at_door"`, and `manifest_items` with `size: "small"`.

**Rationale**:
- `size: "small"` on manifest items ensures Uber assigns a motorcycle courier (Uber Moto), not a car.
- `dropoff_phone_number` must be E.164 format (e.g., `+5493515551234`).
- `dropoff_notes` carries apartment/floor/reference instructions from `CustomerAddress.ApartmentNotes`.
- Response includes `uuid` (external_delivery_id), `tracking_url`, `status`, and `fee`.
- If the `quote_id` is expired, the API returns a specific error that we must catch to auto-renew.

**Alternatives Considered**:
- **No quote_id (direct delivery)**: Not supported by Uber Direct — quote is required. N/A.

---

## R-005: Uber Webhook Signature Verification

**Decision**: Validate `X-Uber-Signature` header using HMAC-SHA256 computed over the raw request body bytes with the `UBER_WEBHOOK_SECRET` as key. Use `crypto/hmac` and `crypto/subtle.ConstantTimeCompare` for timing-safe comparison.

**Rationale**:
- Uber signs webhook payloads using HMAC-SHA256 on the raw unparsed body.
- The header contains the hex-encoded digest.
- Must read raw body bytes before any JSON parsing.
- Timing-safe comparison prevents timing side-channel attacks.
- Webhook events include `event_type` (e.g., `delivery.status.changed`) and `meta.resource_id` (external_delivery_id).

**Alternatives Considered**:
- **IP allowlisting**: Uber IPs can change. Insufficient alone. Rejected.

---

## R-006: E.164 Phone Number Normalization

**Decision**: WhatsApp `wa_id` values arrive as digit-only strings (e.g., `5493515551234`). Normalize by prepending `+` to form E.164 (`+5493515551234`).

**Rationale**:
- WhatsApp Cloud API delivers `wa_id` without the `+` prefix.
- Uber Direct requires E.164 with the `+` prefix.
- Simple string operation: `"+" + waID`.
- Already validated as numeric digits by WhatsApp platform.

---

## R-007: Quote Expiration Handling (15-Minute Rule)

**Decision**: Store `quote_id` and `quoted_at` timestamp on the order. Before dispatch, check if `time.Since(quoted_at) > 14*time.Minute` (1-minute safety margin). If expired (or if Uber returns a quote-expired error), transparently fetch a new quote and retry dispatch once.

**Rationale**:
- Uber Direct quotes are valid for approximately 15 minutes.
- Using a 14-minute threshold provides safety margin.
- Retry logic: if `CreateDelivery` fails with a quote-expired error, call `QuoteDelivery` again, update the order's quote data, and retry `CreateDelivery` once.
- If the retry also fails, surface the error for operational handling.

---

## R-008: GeocodingProvider Port Design (Hexagonal Architecture)

**Decision**: Define `GeocodingProvider` as a new output port interface in `ports/output/geocoding_provider.go` with a single method `ValidateAndGeocode(ctx, streetAndNumber) (*domain.GeocodedResult, error)`. The adapter in `adapters/output/geocoding/` implements this using Google Maps HTTP API.

**Rationale**:
- Follows the existing project pattern where each external service has its own port interface (see `DeliveryProvider`, `PaymentProvider`).
- The domain layer returns `domain.GeocodedResult` — a pure value object with no HTTP/JSON awareness.
- The adapter handles all HTTP communication, JSON parsing, and mapping to domain types.
- A `MockGeocodingAdapter` is provided for local development when `USE_MOCK_PROVIDERS=true`.

---

## R-009: CustomerAddress Domain Entity Extension

**Decision**: Extend the existing `CustomerAddress` entity with new fields: `RawInput`, `FormattedAddress`, `Neighborhood`, `ZipCode`. Change `City` default from `"CABA"` to `"Córdoba"`.

**Rationale**:
- `RawInput`: preserves the exact text the customer typed in WhatsApp before normalization.
- `FormattedAddress`: the official Google Maps formatted address string.
- `Neighborhood`: extracted from `address_components` with type `neighborhood` or `sublocality`.
- `ZipCode`: extracted from `address_components` with type `postal_code`.
- The existing `StreetAndNumber`, `ApartmentNotes`, `Lat`, `Lng`, `IsDefault` fields are reused as-is.
- GORM auto-migration will add the new columns.

---

## R-010: Existing Codebase Integration Points

**Decision**: Integrate with minimal disruption by:
1. Extending `AddressUseCase` to inject `GeocodingProvider` and call it in `SaveAddress`.
2. Extending the `DeliveryProvider` port with `CancelDelivery` method.
3. Creating a new `DeliveryUseCase` that orchestrates quoting + dispatch with quote renewal.
4. Adding new domain types (`GeocodedResult`, `DeliveryQuote`, `DeliveryRequest`, `DeliveryDispatchResult`) alongside existing domain entities.
5. Modifying `app.go` to wire the new `GeocodingProvider` adapter and extended providers.

**Rationale**:
- The project already has `DeliveryProvider`, `MockDeliveryProvider`, and `UberDirectProvider` stubs — these need to be enhanced, not replaced.
- The `WhatsAppAgentUseCase` already handles `guardar_direccion_cliente` and `cotizar_envio` tool calls — these need to be updated to flow through the enhanced use cases.
- The `Order` entity already has `ExternalDeliveryID`, `TrackingURL`, and `DeliveryCost` fields.
