# Research & Technical Decisions: Topping Price Updates

**Feature**: `001-update-topping-prices`  
**Date**: 2026-10-16  

## Technical Decisions

### Decision 1: HTTP Endpoint Path and Method

- **Decision**: Expose `PATCH /api/toppings/:id/price` protected by `middleware.Protected()`.
- **Rationale**:
  - `PATCH` is the semantically accurate HTTP verb for partial updates to resource attributes.
  - Sub-resource path `/api/toppings/:id/price` explicitly limits scope to price modifications, separating financial/pricing updates from general catalog mutations (e.g., renaming a topping, toggling active status).
  - Maintains strict consistency with existing routes in `adapters/input/http/routes/url_mappings.go` (e.g., `Patch("/orders/:id/cancel", ...)`).
- **Alternatives Considered**:
  - `PUT /api/toppings/:id`: Rejected because `PUT` implies replacing the entire topping entity (requiring `name`, `is_active`, etc.), whereas the business requirement allows updating only prices, optionally partial.
  - `PATCH /api/toppings/:id`: Rejected as an endpoint solely dedicated to prices because it could collide with or complicate broader topping editing permissions. A dedicated price sub-route provides clearer auditing and access boundaries.

---

### Decision 2: DTO Representation and Nullability

- **Decision**: Create a dedicated DTO `UpdateToppingPriceRequest` in `adapters/input/http/dto/catalog_dto.go` with pointer fields:
  ```go
  type UpdateToppingPriceRequest struct {
      SinglePrice *float64 `json:"single_price" validate:"omitempty,min=0"`
      ComboPrice  *float64 `json:"combo_price" validate:"omitempty,min=0"`
  }
  ```
- **Rationale**:
  - Using `*float64` allows distinguishing between an omitted field (`nil`) and an explicit zero value (`0.0`), satisfying requirement FR-003 and the edge case for $0.00 promotional toppings.
  - Handler-level validation checks that at least one of `SinglePrice` or `ComboPrice` is non-nil before delegating to the use case.
  - Leveraging `utils.ValidateStruct` ensures negative values (< 0) are blocked before reaching domain or persistence logic.
- **Alternatives Considered**:
  - Reusing existing `UpdateToppingRequest`: While `UpdateToppingRequest` already has `SinglePrice` and `ComboPrice`, it also exposes `Name` and `IsActive`. Having a dedicated `UpdateToppingPriceRequest` enforces single-responsibility and avoids unintended modifications to other fields.

---

### Decision 3: Hexagonal Architecture Layer Responsibilities

- **Decision**:
  - **Domain** (`core/domain/topping.go`): No schema change needed. The `Topping` entity already has `SinglePrice float64` and `ComboPrice float64`.
  - **Output Port** (`ports/output/topping_repository.go`): Existing `GetByID(id uint) (*domain.Topping, error)` and `Update(topping *domain.Topping) error` methods are fully sufficient.
  - **Use Case** (`core/usecase/catalog_usecase.go`): Add method:
    ```go
    func (u *CatalogUseCase) UpdateToppingPrices(id uint, singlePrice, comboPrice *float64) (*domain.Topping, error)
    ```
    Retrieves the existing topping by ID via `repo.GetByID`. Returns a specific sentinel or not found error if absent. Updates the modified fields and calls `repo.Update`.
  - **Input Adapter (Handler)** (`adapters/input/http/handlers/catalog_handler.go`):
    Add `UpdateToppingPrices(c *fiber.Ctx) error`.
    Parses path parameter `:id`, binds request body, validates input, calls use case, and maps errors to standard HTTP status codes (`400`, `401`, `404`, `500`).
  - **Route Mapping** (`adapters/input/http/routes/url_mappings.go`):
    Register `micro.Patch("/toppings/:id/price", middleware.Protected(), catalogH.UpdateToppingPrices)`.
- **Rationale**: Adheres strictly to the ports-and-adapters architecture established across the `canerollss-backend` codebase.
- **Alternatives Considered**:
  - Updating directly in GORM via `db.Model(&Topping{}).Where("id = ?", id).Updates(...)`: Rejected because it bypasses the domain entity and does not return the updated model unless a reload is executed, breaking domain isolation.

---

### Decision 4: Concurrency and Validation Boundaries

- **Decision**:
  - In-memory validations validate non-negativity and presence of at least one price.
  - GORM `r.db.Save(topping)` or `Updates` persists the entity atomically.
  - Negative numbers are caught by `validator/v10` tags (`min=0`).
  - If a topping ID does not exist, GORM returns `gorm.ErrRecordNotFound`, translated by the use case / handler into HTTP `404 Not Found`.
- **Rationale**: Minimal complexity, standard Go idioms, and consistent with existing `CustomerHandler` and `OrderHandler` error handling patterns.
