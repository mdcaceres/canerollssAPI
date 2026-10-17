# Quickstart Validation Guide: Topping Price Updates

**Feature**: `001-update-topping-prices`  
**Date**: 2026-10-16  

This guide provides runnable scenarios to validate the topping price update endpoint end-to-end.

## Prerequisites

1. The backend server is running locally (e.g. `go run main.go`).
2. An authenticated user exists with valid credentials to obtain a JWT token.
3. At least one topping exists in the database (or created via `POST /api/toppings`).

---

## Step 1: Obtain Auth Token

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "yourpassword"}'
```

Save the returned token:
```bash
export TOKEN="<jwt_token>"
```

---

## Step 2: Validate Scenario 1 - Update Both Prices (P1)

Update topping with ID `1` to `single_price: 2.50` and `combo_price: 1.75`:

```bash
curl -X PATCH http://localhost:8080/api/toppings/1/price \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "single_price": 2.50,
    "combo_price": 1.75
  }'
```

**Expected Response**: `200 OK`
```json
{
  "id": 1,
  "name": "Nutella",
  "single_price": 2.5,
  "combo_price": 1.75,
  "is_active": true,
  "created_at": "...",
  "updated_at": "..."
}
```

---

## Step 3: Validate Scenario 2 - Partial Update (P2)

Update only the `combo_price` to `1.90` without modifying `single_price`:

```bash
curl -X PATCH http://localhost:8080/api/toppings/1/price \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "combo_price": 1.90
  }'
```

**Expected Response**: `200 OK`
```json
{
  "id": 1,
  "name": "Nutella",
  "single_price": 2.5,
  "combo_price": 1.9,
  "is_active": true,
  "created_at": "...",
  "updated_at": "..."
}
```

---

## Step 4: Validate Scenario 3 - Validation Error on Negative Price (P3)

Send a negative amount for `single_price`:

```bash
curl -X PATCH http://localhost:8080/api/toppings/1/price \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "single_price": -0.50
  }'
```

**Expected Response**: `400 Bad Request`
```json
{
  "error": "single_price must be greater than or equal to 0"
}
```

---

## Step 5: Validate Scenario 4 - Nonexistent Topping (P3)

Attempt to update a non-existing topping (e.g., ID `99999`):

```bash
curl -X PATCH http://localhost:8080/api/toppings/99999/price \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "single_price": 3.00
  }'
```

**Expected Response**: `404 Not Found`
```json
{
  "error": "topping not found"
}
```

---

## Step 6: Validate Scenario 5 - Empty Payload Error

Send a body with neither price specified:

```bash
curl -X PATCH http://localhost:8080/api/toppings/1/price \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Expected Response**: `400 Bad Request`
```json
{
  "error": "at least one price (single_price or combo_price) must be provided"
}
```
