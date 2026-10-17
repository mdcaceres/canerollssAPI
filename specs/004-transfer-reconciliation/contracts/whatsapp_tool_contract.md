# WhatsApp Agent Tool Contract: Transfer Verification

**Feature**: `004-transfer-reconciliation`  
**Tool Name**: `verificar_estado_pago`  

---

## 1. Tool Declaration (GenAI Function Declaration)

```json
{
  "name": "verificar_estado_pago",
  "description": "Verifica automáticamente si la transferencia bancaria o de billetera virtual ingresó a la cuenta de Mercado Pago (Alias Caneroll.mp). Invocala cuando el cliente diga 'ya pagué', 'ya transferí' o pase un comprobante.",
  "parameters": {
    "type": "OBJECT",
    "properties": {
      "pedido_id": {
        "type": "INTEGER",
        "description": "ID numérico del pedido a verificar (opcional si existe un pedido pendiente activo en la sesión)."
      },
      "numero_operacion": {
        "type": "STRING",
        "description": "Número de operación, comprobante o transferencia indicado por el cliente (opcional, para desambiguación)."
      },
      "nombre_titular": {
        "type": "STRING",
        "description": "Nombre del titular de la cuenta bancaria de origen (opcional, para desambiguación)."
      }
    }
  }
}
```

---

## 2. Tool Execution Output Schema

```json
{
  "status": "MATCHED",
  "pedido_id": 105,
  "monto": 8500.00,
  "operacion_id": "12345678901",
  "pagador": "Juan Pérez",
  "mensaje": "Transferencia por $8.500 recibida y conciliada con éxito. ¡El pedido ya está confirmado y en preparación!"
}
```

### Alternative Status Outputs

- **`NOT_FOUND_YET`**:
  ```json
  {
    "status": "NOT_FOUND_YET",
    "pedido_id": 105,
    "monto_esperado": 8500.00,
    "mensaje": "Aún no visualizamos la transferencia de $8.500 en nuestra cuenta de Mercado Pago. A veces las transferencias bancarias demoran 1 o 2 minutos en impactar. Por favor aguardá un instante y volvé a consultarme."
  }
  ```
- **`AMBIGUOUS_MULTIPLE_MATCHES`**:
  ```json
  {
    "status": "AMBIGUOUS_MULTIPLE_MATCHES",
    "pedido_id": 105,
    "monto_esperado": 8500.00,
    "mensaje": "Detectamos más de una transferencia por $8.500 en este horario. ¿Podrías indicarme el número de operación del comprobante o el nombre del titular de la cuenta de origen para confirmártelo al instante?"
  }
  ```
- **`AMOUNT_MISMATCH`**:
  ```json
  {
    "status": "AMOUNT_MISMATCH",
    "pedido_id": 105,
    "monto_esperado": 8500.00,
    "monto_recibido": 4000.00,
    "mensaje": "La transferencia registrada es por $4.000 pero el total de tu pedido es de $8.500. Por favor verificá el monto restante."
  }
  ```
