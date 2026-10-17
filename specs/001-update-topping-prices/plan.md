# Implementation Plan: Topping Price Updates

**Branch**: `001-update-topping-prices` | **Date**: 2026-10-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-update-topping-prices/spec.md`

## Summary

Implement a dedicated HTTP endpoint `PATCH /api/toppings/:id/price` to allow updating individual and combo prices of cinnamon roll toppings. The solution extends the existing Hexagonal Architecture:
- Adding a dedicated DTO `UpdateToppingPriceRequest` with pointer floats to support partial updates and zero-price values.
- Introducing `UpdateToppingPrices` in `CatalogUseCase` to fetch the entity via the existing `ToppingRepository`, update modified values, and persist atomically.
- Exposing the handler method in `CatalogHandler` and routing it under `/api/toppings/:id/price` with JWT authentication middleware.

## Technical Context

**Language/Version**: Go 1.26 (compatible with Go 1.23+)

**Primary Dependencies**:
- Fiber v2 (`github.com/gofiber/fiber/v2 v2.52.13`)
- GORM (`gorm.io/gorm v1.31.1`) with MySQL driver (`gorm.io/driver/mysql v1.6.0`)
- Validator (`github.com/go-playground/validator/v10 v10.30.3`)
- JWT (`github.com/golang-jwt/jwt/v5 v5.3.1`)

**Storage**: MySQL / MariaDB relational database managed via GORM.

**Testing**: Go standard testing package (`go test ./...`), HTTP handler tests with Fiber test utilities (`app.Test`).

**Target Platform**: Linux / Containerized cloud runtime or local Windows development environment.

**Project Type**: RESTful Web Service / Backend API.

**Performance Goals**: < 100ms response time for price updates; instant cache/read availability.

**Constraints**:
- Must preserve Hexagonal Architecture layer separation (domain, use case, ports, adapters).
- Must enforce non-negative prices (≥ 0.00) while allowing $0.00 for promotional/free toppings.
- Must reject updates to nonexistent toppings with HTTP 404.

**Scale/Scope**: Single endpoint addition touching 4 existing files and leveraging existing repository methods without database migrations.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Hexagonal Architecture Compliance**: PASS. Domain logic lives in `core/domain` and `core/usecase`. Infrastructure details remain in `adapters/input/http` and `adapters/output/repository`. No direct database queries in handlers.
- **Contract & Validation Gate**: PASS. Validations are enforced at the DTO boundary via `validator/v10` and domain logic in the use case.
- **Security & Authorization Gate**: PASS. Endpoint is protected using existing `middleware.Protected()`.
- **Backward Compatibility**: PASS. Existing `Topping` schema, existing endpoints (`GET /api/toppings`, `POST /api/toppings`), and order calculation logic remain backwards-compatible.

## Project Structure

### Documentation (this feature)

```text
specs/001-update-topping-prices/
├── plan.md              # Implementation plan (/speckit-plan output)
├── research.md          # Phase 0 technical decisions (/speckit-plan output)
├── data-model.md        # Phase 1 domain and DTO models (/speckit-plan output)
├── quickstart.md        # Phase 1 end-to-end verification guide (/speckit-plan output)
├── contracts/           # Phase 1 API contracts (/speckit-plan output)
│   └── update-topping-price.json
└── checklists/
    └── requirements.md  # Requirements quality checklist
```

### Source Code (repository layout)

```text
canerollss-backend/
├── core/
│   ├── domain/
│   │   └── topping.go                              # Existing Topping entity
│   └── usecase/
│       └── catalog_usecase.go                      # Add UpdateToppingPrices method
├── ports/
│   └── output/
│       └── topping_repository.go                   # Existing ToppingRepository interface (GetByID, Update)
├── adapters/
│   ├── input/http/
│   │   ├── dto/
│   │   │   └── catalog_dto.go                      # Add UpdateToppingPriceRequest DTO
│   │   ├── handlers/
│   │   │   └── catalog_handler.go                  # Add UpdateToppingPrices handler
│   │   └── routes/
│   │       └── url_mappings.go                     # Register PATCH /toppings/:id/price
│   └── output/
│       └── repository/
│           └── topping_repository.go               # Existing GORM repository implementation
└── utils/
    └── validations.go                              # Existing validation helper
```

**Structure Decision**: Hexagonal Architecture. The changes integrate seamlessly into the existing packages without creating unnecessary new abstractions or packages.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| :--- | :--- | :--- |
| None | N/A | Feature strictly adheres to existing patterns and architecture. |
