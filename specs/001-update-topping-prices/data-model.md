# Data Model: Topping Price Updates

**Feature**: `001-update-topping-prices`  
**Date**: 2026-10-16  

## Entities

### Topping

The core domain entity representing a topping or extra flavor that can be added to cinnamon rolls.

```text
+-------------------------------------------------------+
|                       Topping                         |
+-------------------------------------------------------+
| ID          : uint           (Primary Key)            |
| Name        : string         (unique, max 50)         |
| SinglePrice : float64        (not null, >= 0.0)       |
| ComboPrice  : float64        (not null, >= 0.0)       |
| IsActive    : bool           (default: true)          |
| CreatedAt   : time.Time                               |
| UpdatedAt   : time.Time                               |
| DeletedAt   : gorm.DeletedAt (soft delete index)      |
+-------------------------------------------------------+
```

### Fields Description

| Field | Type | Required | Constraints | Description |
| :--- | :--- | :---: | :--- | :--- |
| `ID` | `uint` | Yes | Primary Key, Auto-increment | Unique identifier of the topping in the database. |
| `Name` | `string` | Yes | Unique, length 2–50 | Commercial name of the topping (e.g., "Nutella", "Frutos Rojos"). |
| `SinglePrice` | `float64` | Yes | `>= 0.0` | Charge added when ordered with an individual cinnamon roll. |
| `ComboPrice` | `float64` | Yes | `>= 0.0` | Charge added when ordered as part of a combo. |
| `IsActive` | `bool` | Yes | Default: `true` | Indicates whether the topping is active for customer ordering. |
| `CreatedAt` | `time.Time` | Yes | System-managed | Timestamp when the record was created. |
| `UpdatedAt` | `time.Time` | Yes | System-managed | Timestamp updated automatically whenever fields change. |
| `DeletedAt` | `gorm.DeletedAt`| No | Index | Soft deletion timestamp. |

---

## Data Transfer Objects (DTO)

### UpdateToppingPriceRequest

Payload sent by the client when updating one or both prices:

```go
type UpdateToppingPriceRequest struct {
    SinglePrice *float64 `json:"single_price" validate:"omitempty,min=0"`
    ComboPrice  *float64 `json:"combo_price" validate:"omitempty,min=0"`
}
```

#### Invariants & Validation Rules

1. **Existence**: Target `ID` must reference an active, non-deleted `Topping` record. If not found, a `404 Not Found` response is returned.
2. **At Least One Field**: At least one of `SinglePrice` or `ComboPrice` must be provided (non-nil). If both are `nil`, a `400 Bad Request` response is returned.
3. **Non-negativity**: Any provided price must satisfy `price >= 0.00`. Negative values return `400 Bad Request`.
4. **Zero-price Allowance**: Values of `0.00` are valid (used for complimentary or promotional toppings).
5. **Timestamp Mutation**: Upon successful update, `UpdatedAt` is updated to current UTC time.
