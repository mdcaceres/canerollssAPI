# Feature Specification: Topping Price Updates

**Feature Branch**: `001-update-topping-prices`

**Created**: 2026-10-16

**Status**: Draft

**Input**: User description: "Desarrollar un endpoint de cambio de precios para los toppings de los rolls de canela tanto individuales como en combo, siguiendo los patroes del repositorio @c:\Users\mdcac\canerollss\canerollss-backend\core\domain\topping.go:L12-L13"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Full Topping Price Update (Priority: P1)

As a Cinnamon Rolls store manager or administrator, I want to simultaneously update both the single price and the combo price of an existing topping so that the catalog reflects new ingredient costs or promotional strategies.

**Why this priority**: Core functionality with direct operational impact; it ensures the business can keep its pricing up to date across both sales formats (individual consumption and combos), preventing revenue loss or billing discrepancies.

**Independent Test**: Can be tested by updating an existing topping with new amounts for both single and combo prices, then verifying that the topping immediately returns the updated values in catalog lookups and sales calculations.

**Acceptance Scenarios**:

1. **Given** an existing topping with single price $1.50 and combo price $1.00, **When** an authorized user submits an update with single price $2.00 and combo price $1.25, **Then** the system confirms the update and records both new prices.
2. **Given** an existing topping, **When** updated with valid new prices, **Then** any new orders or catalog inquiries immediately reflect the updated prices without requiring a system restart.

---

### User Story 2 - Partial Price Update (Priority: P2)

As a store administrator, I want to adjust either only the single price or only the combo price of a specific topping without having to re-enter the unchanged price value.

**Why this priority**: Provides operational flexibility for targeted promotions or margin adjustments that only affect one of the two sales modalities.

**Independent Test**: Update only one price field (e.g., combo price only) and verify that only the targeted price changes while the untouched price retains its previous value.

**Acceptance Scenarios**:

1. **Given** an existing topping with single price $2.00 and combo price $1.25, **When** a request is submitted changing only the combo price to $1.50, **Then** the combo price is updated to $1.50 while the single price remains unchanged at $2.00.
2. **Given** an existing topping with single price $2.00 and combo price $1.25, **When** a request is submitted changing only the single price to $2.50, **Then** the single price is updated to $2.50 while the combo price remains unchanged at $1.25.

---

### User Story 3 - Price Validation and Error Controls (Priority: P3)

As a sales management system, I want to validate that submitted price values are valid and that the target topping exists, preventing accounting inconsistencies or negative pricing due to human error.

**Why this priority**: Guarantees catalog data integrity and prevents financial losses resulting from typographical mistakes during price configuration.

**Independent Test**: Attempt price updates with negative values, non-numeric payloads, or nonexistent topping identifiers, confirming that the system rejects the operation with clear, descriptive messages.

**Acceptance Scenarios**:

1. **Given** a price update request, **When** a negative value is provided for single price or combo price (e.g., -$0.50), **Then** the system rejects the request stating that prices must be greater than or equal to zero.
2. **Given** a topping identifier that does not exist in the catalog, **When** a price update is requested, **Then** the system rejects the request indicating that the topping was not found.
3. **Given** an update request where neither single price nor combo price is provided, **When** the request is processed, **Then** the system rejects the request stating that at least one price must be provided.

---

### Edge Cases

- **Zero Price ($0.00)**: If a topping is offered complimentary or as part of a promotional giveaway, the system must allow setting $0.00 for either single price, combo price, or both.
- **Inactive Topping**: If a topping is marked as inactive in the catalog, the system must still allow updating its prices so that pricing can be prepared prior to reactivation.
- **Excessive Decimal Precision**: If inputs contain more than two decimal places (e.g., 1.259), the system must enforce standard currency rounding rules (two decimal places).
- **Concurrent Modifications**: If two administrators update the prices for the same topping concurrently, the system must atomically persist the last confirmed transaction.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow authenticated and authorized users to modify the prices of an existing topping using its unique identifier.
- **FR-002**: The system MUST support simultaneous updates of both single price (`SinglePrice`) and combo price (`ComboPrice`).
- **FR-003**: The system MUST allow partial updates, accepting changes to only one price field (`SinglePrice` or `ComboPrice`) while preserving the existing value of the other.
- **FR-004**: The system MUST validate that all provided price values are numeric values greater than or equal to zero (≥ 0.00).
- **FR-005**: The system MUST reject any update request if the topping identifier does not match any existing record in the catalog.
- **FR-006**: The system MUST reject update requests that contain neither single price nor combo price.
- **FR-007**: The system MUST return the updated topping details (including both prices and the update timestamp) upon successful modification.
- **FR-008**: New sales orders created after a price modification MUST calculate line item and total amounts using the updated prices.

### Key Entities

- **Topping (Cinnamon Roll Topping/Flavor)**:
  - *Identifier*: Unique numeric catalog identifier.
  - *Name*: Descriptive name of the flavor or additional ingredient (e.g., "Nutella", "Dulce de Leche", "Pecans", "Berries").
  - *Single Price*: Monetary amount charged when the topping is added to an individual roll.
  - *Combo Price*: Monetary amount charged when the topping is included within a combo package.
  - *Active Status*: Availability flag for current menu sales.
  - *Timestamps*: Records of creation and last modification date/time.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Updated topping prices are immediately available for new sales and catalog inquiries in under 1 second following confirmation.
- **SC-002**: 100% of invalid price requests (negative amounts or malformed numbers) are rejected before any database alteration occurs.
- **SC-003**: 100% of orders placed immediately after a price change accurately reflect the updated pricing without discrepancies in total calculations.
- **SC-004**: Store administrators can complete a price change for any topping in under 15 seconds through the management interface.

## Assumptions

- Users performing this action have valid catalog administrative or editing permissions.
- The currency unit is consistent across the application and does not require real-time currency conversion.
- Modifying prices does not retroactively change existing, closed, or in-flight orders; it applies exclusively to new transactions.
- The toppings catalog is pre-existing and currently supports listing and creation operations.
