# Implementation Plan: Geographic Address Validation & Home Delivery Logistics

**Branch**: `005-geo-delivery-validation` | **Date**: 2026-10-17 | **Spec**: [spec.md](spec.md)

## Summary

Validate customer addresses with Google Maps before saving them, quote coordinate-based delivery with Uber Direct, let authenticated store staff dispatch paid orders, and process signed courier status webhooks. The WhatsApp agent can save addresses and quote costs but has no courier dispatch tool.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Fiber v2, GORM, Google Gemini SDK, standard-library HTTP and crypto packages

**Storage**: MySQL in production; SQLite in isolated local tests; GORM AutoMigrate

**Testing**: Go unit, provider contract, HTTP handler, and end-to-end tests

**Target Platform**: Go HTTP service

**Project Type**: Single backend service using Clean Architecture and ports/adapters

**Performance Goals**: Address feedback within 5 seconds; tracking notification within 10 seconds of dispatch; webhook status update within 3 seconds

**Constraints**: Córdoba Capital only; exact street number and coordinates required; manual operator dispatch by default; HMAC-SHA256 webhook verification

## Constitution Check

`.specify/memory/constitution.md` contains an unfilled template and defines no additional project constraints. The feature follows the existing domain, use case, port, and adapter boundaries.

## Project Structure

```text
core/domain/                         # address, order, and delivery values
core/usecase/                        # address, quote, dispatch, webhook orchestration
ports/output/                         # geocoding, courier, repositories, notification
adapters/output/geocoding/            # Google Maps and mock
adapters/output/delivery/             # Uber Direct and mock
adapters/output/repository/           # GORM persistence
adapters/output/whatsapp/             # customer notification
adapters/input/http/handlers/         # operator dispatch and signed webhook
adapters/input/http/routes/            # route registration
app/app.go                             # dependency composition
tests/                                 # end-to-end flow
```

## Design Notes

- Provider adapters own HTTP, OAuth, JSON mapping, and external error decoding. The core depends on output ports.
- An order stores quote ID and timestamps so dispatch can renew near-expiry quotes and retry once after an expired-quote response.
- The operator route requires an authenticated employee or admin session and a paid order with a validated destination.
- Uber sends `event.delivery_status` webhooks with `delivery_id` and `status`; signature verification uses the raw request body.
- Mock providers are selected by `USE_MOCK_PROVIDERS` for local operation without credentials.
