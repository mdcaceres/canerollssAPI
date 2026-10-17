package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/input"
	"canerollss/ports/output"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"
)

type WhatsAppAgentUseCase struct {
	client           *genai.Client
	orderUseCase     *OrderUseCase
	addressUseCase   *AddressUseCase
	reconciliationUC *PaymentReconciliationUseCase
	customerRepo     output.CustomerRepository
	toppingRepo      output.ToppingRepository
	cashRepo         output.CashRegisterRepository
	deliveryPort     output.DeliveryProvider
	paymentPort      output.PaymentProvider
	apiKey           string
	promptBuilder    *AgentPromptBuilder
}

func NewWhatsAppAgentUseCase(
	orderUC *OrderUseCase,
	addrUC *AddressUseCase,
	custRepo output.CustomerRepository,
	topRepo output.ToppingRepository,
	cashRepo output.CashRegisterRepository,
	delivery output.DeliveryProvider,
	payment output.PaymentProvider,
	reconcile ...*PaymentReconciliationUseCase,
) *WhatsAppAgentUseCase {
	apiKey := os.Getenv("GEMINI_API_KEY")
	var client *genai.Client
	if apiKey != "" && apiKey != "mock_gemini_key" {
		ctx := context.Background()
		c, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			log.Printf("[WhatsAppAgent] Warning: could not initialize genai client: %v", err)
		} else {
			client = c
		}
	}

	builder, err := NewAgentPromptBuilder()
	if err != nil {
		log.Printf("[WhatsAppAgent] Warning: could not initialize AgentPromptBuilder: %v", err)
	}

	var reconUC *PaymentReconciliationUseCase
	if len(reconcile) > 0 && reconcile[0] != nil {
		reconUC = reconcile[0]
	} else if orderUC != nil && cashRepo != nil && payment != nil {
		reconUC = NewPaymentReconciliationUseCase(orderUC.orderRepo, cashRepo, payment)
	}

	return &WhatsAppAgentUseCase{
		client:           client,
		orderUseCase:     orderUC,
		addressUseCase:   addrUC,
		reconciliationUC: reconUC,
		customerRepo:     custRepo,
		toppingRepo:      topRepo,
		cashRepo:         cashRepo,
		deliveryPort:     delivery,
		paymentPort:      payment,
		apiKey:           apiKey,
		promptBuilder:    builder,
	}
}

// BuildSessionContext gathers real-time session state for prompt rendering and tool gating
func (a *WhatsAppAgentUseCase) BuildSessionContext(customerPhone, profileName string) AgentSessionContext {
	loc := time.FixedZone("ART", -3*3600)
	now := time.Now().In(loc)
	days := []string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}
	currentTime := fmt.Sprintf("%s %02d:%02d", days[now.Weekday()], now.Hour(), now.Minute())

	custName := strings.TrimSpace(profileName)
	if custName == "" {
		custName = "Cliente"
	}

	isRegisterOpen := false
	var activeRegID uint
	if a.cashRepo != nil {
		if reg, err := a.cashRepo.GetActive(); err == nil && reg != nil && reg.Status == domain.RegisterStatusOpen {
			isRegisterOpen = true
			activeRegID = reg.ID
		}
	}

	hasSavedAddresses := false
	if a.addressUseCase != nil {
		if addrs, err := a.addressUseCase.GetCustomerAddresses(customerPhone); err == nil && len(addrs) > 0 {
			hasSavedAddresses = true
		}
	}

	pendingOrderID := ""
	if a.customerRepo != nil && a.orderUseCase != nil && activeRegID > 0 {
		if cust, err := a.customerRepo.FindByWhatsapp(customerPhone); err == nil && cust != nil {
			if pendingOrder, err := a.orderUseCase.GetPendingOrderByCustomer(activeRegID, cust.ID); err == nil && pendingOrder != nil {
				pendingOrderID = fmt.Sprintf("%d", pendingOrder.ID)
			}
		}
	}

	return AgentSessionContext{
		CurrentTime:        currentTime,
		CustomerName:       custName,
		CustomerPhone:      customerPhone,
		IsCashRegisterOpen: isRegisterOpen,
		HasSavedAddresses:  hasSavedAddresses,
		PendingOrderID:     pendingOrderID,
	}
}

// ResolveToolConfig restricts available functions at the protocol level based on cash register status
func (a *WhatsAppAgentUseCase) ResolveToolConfig(isRegisterOpen bool) *genai.ToolConfig {
	if !isRegisterOpen {
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
				AllowedFunctionNames: []string{
					"consultar_catalogo",
					"verificar_estado_pago",
				},
			},
		}
	}

	return &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode: genai.FunctionCallingConfigModeAny,
			AllowedFunctionNames: []string{
				"consultar_catalogo",
				"obtener_direcciones_cliente",
				"guardar_direccion_cliente",
				"cotizar_envio",
				"crear_pedido_con_link_pago",
				"verificar_estado_pago",
			},
		},
	}
}

func (a *WhatsAppAgentUseCase) getSystemInstruction(sessionCtx AgentSessionContext) string {
	if a.promptBuilder != nil {
		instruction, err := a.promptBuilder.BuildSystemInstruction(sessionCtx)
		if err == nil {
			return instruction
		}
		log.Printf("[WhatsAppAgent] Warning: error building system instruction: %v", err)
	}

	return `Sos "Roli", la asistente virtual de CaneRolls, la mejor tienda de roles de canela artesanales recién horneados en Córdoba. Tu tono es rioplatense cordobés amable, cercano y eficiente.`
}

func (a *WhatsAppAgentUseCase) getToolsDeclarations() []*genai.Tool {
	return []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "consultar_catalogo",
					Description: "Obtiene la lista en tiempo real de todos los sabores de roles de canela activos en la tienda con sus precios individuales y en combo.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
					},
				},
				{
					Name:        "obtener_direcciones_cliente",
					Description: "Obtiene la lista de direcciones guardadas por el cliente actual asociadas a su sesión de WhatsApp.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
					},
				},
				{
					Name:        "guardar_direccion_cliente",
					Description: "Guarda una nueva dirección de entrega para el cliente actual.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"calle_y_altura": {
								Type:        genai.TypeString,
								Description: "Calle y número de la dirección (ej: 'Av. Corrientes 1234').",
							},
							"piso_depto_referencia": {
								Type:        genai.TypeString,
								Description: "Piso, departamento, timbre o referencias adicionales.",
							},
							"alias": {
								Type:        genai.TypeString,
								Description: "Nombre identificatorio de la dirección (ej: 'Casa', 'Trabajo').",
							},
						},
						Required: []string{"calle_y_altura"},
					},
				},
				{
					Name:        "cotizar_envio",
					Description: "Cotiza el costo de entrega hacia una de las direcciones guardadas del cliente antes de confirmar el pedido.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"address_id": {
								Type:        genai.TypeInteger,
								Description: "ID numérico de la dirección del cliente.",
							},
						},
						Required: []string{"address_id"},
					},
				},
				{
					Name:        "crear_pedido_con_link_pago",
					Description: "Crea el pedido formal en estado PENDING_PAYMENT con los ítems confirmados, asocia costo de envío si corresponde y genera el link de pago.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"items": {
								Type:        genai.TypeArray,
								Description: "Lista de productos seleccionados.",
								Items: &genai.Schema{
									Type: genai.TypeObject,
									Properties: map[string]*genai.Schema{
										"sabor": {
											Type:        genai.TypeString,
											Description: "Nombre del sabor (ej: 'Clasico', 'Dulce de leche', 'Avellana', 'Oreo', 'Pistacho premium').",
										},
										"formato": {
											Type:        genai.TypeString,
											Description: "'individual' (1 unidad) o 'combo' (2 unidades).",
										},
										"cantidad": {
											Type:        genai.TypeInteger,
											Description: "Cantidad de unidades o combos.",
										},
									},
									Required: []string{"sabor", "formato", "cantidad"},
								},
							},
							"modalidad_entrega": {
								Type:        genai.TypeString,
								Description: "Modalidad: 'retiro' o 'envio'.",
							},
							"address_id": {
								Type:        genai.TypeInteger,
								Description: "ID de la dirección de entrega (requerido si modalidad es 'envio').",
							},
						},
						Required: []string{"items", "modalidad_entrega"},
					},
				},
				{
					Name:        "verificar_estado_pago",
					Description: "Verifica automáticamente si la transferencia bancaria o de billetera virtual ingresó a la cuenta de Mercado Pago (Alias Caneroll.mp). Invocala cuando el cliente diga 'ya pagué', 'ya transferí' o pase un comprobante.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"pedido_id": {
								Type:        genai.TypeInteger,
								Description: "ID numérico del pedido a verificar (opcional si hay un pedido activo pendiente en la sesión).",
							},
							"numero_operacion": {
								Type:        genai.TypeString,
								Description: "Número de operación, comprobante o transferencia indicado por el cliente (opcional).",
							},
							"nombre_titular": {
								Type:        genai.TypeString,
								Description: "Nombre del titular de la cuenta bancaria o billetera de origen (opcional).",
							},
						},
					},
				},
			},
		},
	}
}

// ProcessMessage processes an incoming text batch from a customer
func (a *WhatsAppAgentUseCase) ProcessMessage(
	ctx context.Context,
	sessionCtx AgentSessionContext,
	userPrompt string,
	history []*genai.Content,
) (string, []*genai.Content, error) {
	if a.client == nil {
		// Fallback when live Gemini client is not initialized (dev/mock mode)
		return a.handleFallback(ctx, sessionCtx, userPrompt, history)
	}

	instruction := a.getSystemInstruction(sessionCtx)

	// If cash register is closed, enforce protocol-level restriction with ToolConfig
	var initialToolConfig *genai.ToolConfig
	if !sessionCtx.IsCashRegisterOpen {
		initialToolConfig = a.ResolveToolConfig(false)
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(instruction)},
		},
		Tools:      a.getToolsDeclarations(),
		ToolConfig: initialToolConfig,
	}

	// Prepare contents
	contents := make([]*genai.Content, 0, len(history)+1)
	contents = append(contents, history...)
	userContent := &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{genai.NewPartFromText(userPrompt)},
	}
	contents = append(contents, userContent)

	// Tool execution loop (max 5 iterations)
	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.5-flash-lite"
	}
	modelsToTry := []string{primaryModel, "gemini-3.5-flash-lite", "gemini-3.1-flash-lite", "gemini-3.5-flash"}
	var uniqueModels []string
	seen := make(map[string]bool)
	for _, m := range modelsToTry {
		if !seen[m] {
			seen[m] = true
			uniqueModels = append(uniqueModels, m)
		}
	}

	for i := 0; i < 5; i++ {
		var resp *genai.GenerateContentResponse
		var lastErr error

		for _, m := range uniqueModels {
			resp, lastErr = a.client.Models.GenerateContent(ctx, m, contents, config)
			if lastErr == nil && resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				break
			}
			log.Printf("[WhatsAppAgent] Model %s failed (%v). Retrying with next model...", m, lastErr)
			time.Sleep(200 * time.Millisecond)
		}

		if lastErr != nil || resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			log.Printf("[WhatsAppAgent] All GenAI models failed: %v. Using fallback.", lastErr)
			return a.handleFallback(ctx, sessionCtx, userPrompt, history)
		}

		candidate := resp.Candidates[0]
		contents = append(contents, candidate.Content)

		// Check for function calls
		var functionCalls []*genai.FunctionCall
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
		}

		if len(functionCalls) == 0 {
			// Model responded with final text
			var sb strings.Builder
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					sb.WriteString(part.Text)
				}
			}
			return sb.String(), contents, nil
		}

		// Execute function calls
		responseParts := make([]*genai.Part, 0, len(functionCalls))
		for _, fc := range functionCalls {
			toolResult := a.executeTool(ctx, sessionCtx.CustomerPhone, sessionCtx.CustomerName, fc.Name, fc.Args)
			responseParts = append(responseParts, genai.NewPartFromFunctionResponse(fc.Name, toolResult))
		}

		contents = append(contents, &genai.Content{
			Role:  "user",
			Parts: responseParts,
		})

		// After tool execution, allow natural language response generation
		config.ToolConfig = nil
	}

	return "Aquí tienes la información sobre tu pedido. ¿Deseas algo más?", contents, nil
}

func (a *WhatsAppAgentUseCase) executeTool(
	ctx context.Context,
	customerPhone string,
	profileName string,
	toolName string,
	args map[string]any,
) map[string]any {
	switch toolName {
	case "consultar_catalogo":
		toppings, err := a.toppingRepo.GetAll()
		if err != nil {
			return map[string]any{"error": "No se pudo obtener el catálogo"}
		}
		var active []map[string]any
		for _, t := range toppings {
			if t.IsActive {
				active = append(active, map[string]any{
					"id":           t.ID,
					"nombre":       t.Name,
					"precio_indiv": t.SinglePrice,
					"precio_combo": t.ComboPrice,
				})
			}
		}
		return map[string]any{"sabores_disponibles": active}

	case "obtener_direcciones_cliente":
		addresses, err := a.addressUseCase.GetCustomerAddresses(customerPhone)
		if err != nil {
			return map[string]any{"direcciones": []any{}, "mensaje": "Sin direcciones guardadas"}
		}
		var list []map[string]any
		for _, addr := range addresses {
			list = append(list, map[string]any{
				"id":                    addr.ID,
				"alias":                 addr.Alias,
				"calle_y_altura":        addr.StreetAndNumber,
				"piso_depto_referencia": addr.ApartmentNotes,
				"es_predeterminada":     addr.IsDefault,
			})
		}
		return map[string]any{"direcciones": list}

	case "guardar_direccion_cliente":
		calle, _ := args["calle_y_altura"].(string)
		depto, _ := args["piso_depto_referencia"].(string)
		alias, _ := args["alias"].(string)
		if alias == "" {
			alias = "Casa"
		}
		addr, err := a.addressUseCase.SaveAddress(customerPhone, alias, calle, depto)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{
			"status":     "guardada",
			"address_id": addr.ID,
			"alias":      addr.Alias,
			"direccion":  addr.StreetAndNumber,
		}

	case "cotizar_envio":
		addrIDVal, ok := args["address_id"]
		if !ok {
			return map[string]any{"error": "address_id requerido"}
		}
		var addrID uint
		switch v := addrIDVal.(type) {
		case uint:
			addrID = v
		case int:
			addrID = uint(v)
		case int64:
			addrID = uint(v)
		case uint64:
			addrID = uint(v)
		case float64:
			addrID = uint(v)
		}
		addr, err := a.addressUseCase.GetAddressByID(addrID)
		if err != nil || addr == nil {
			return map[string]any{"error": "Dirección no encontrada"}
		}
		quote, err := a.deliveryPort.QuoteDelivery(ctx, addr)
		if err != nil {
			return map[string]any{"error": fmt.Sprintf("No se pudo cotizar envío: %v", err)}
		}
		return map[string]any{
			"costo_envio":   quote.EstimatedFee,
			"moneda":        quote.Currency,
			"duracion_min":  quote.DurationMin,
			"direccion":     addr.StreetAndNumber,
		}

	case "crear_pedido_con_link_pago":
		return a.executeCrearPedido(ctx, customerPhone, profileName, args)

	case "verificar_estado_pago":
		var orderID uint
		if pedidoIDVal, ok := args["pedido_id"]; ok && pedidoIDVal != nil {
			switch v := pedidoIDVal.(type) {
			case uint:
				orderID = v
			case int:
				orderID = uint(v)
			case int64:
				orderID = uint(v)
			case uint64:
				orderID = uint(v)
			case float64:
				orderID = uint(v)
			}
		}

		if orderID == 0 && a.customerRepo != nil && a.cashRepo != nil {
			if cust, err := a.customerRepo.FindByWhatsapp(customerPhone); err == nil && cust != nil {
				if activeReg, err := a.cashRepo.GetActive(); err == nil && activeReg != nil {
					if pendingOrder, err := a.orderUseCase.GetPendingOrderByCustomer(activeReg.ID, cust.ID); err == nil && pendingOrder != nil {
						orderID = pendingOrder.ID
					}
				}
			}
		}

		if orderID == 0 {
			return map[string]any{"error": "No se encontró ningún pedido pendiente de pago asociado a tu número de teléfono."}
		}

		var numeroOp string
		if opVal, ok := args["numero_operacion"]; ok && opVal != nil {
			switch v := opVal.(type) {
			case string:
				numeroOp = v
			case int:
				numeroOp = strconv.Itoa(v)
			case int64:
				numeroOp = strconv.FormatInt(v, 10)
			case uint:
				numeroOp = strconv.FormatUint(uint64(v), 10)
			case uint64:
				numeroOp = strconv.FormatUint(v, 10)
			case float64:
				numeroOp = strconv.FormatInt(int64(v), 10)
			default:
				numeroOp = fmt.Sprintf("%v", v)
			}
		} else if compVal, ok := args["comprobante"]; ok && compVal != nil {
			numeroOp = fmt.Sprintf("%v", compVal)
		} else if vouchVal, ok := args["voucher"]; ok && vouchVal != nil {
			numeroOp = fmt.Sprintf("%v", vouchVal)
		}

		var nombreTit string
		if titVal, ok := args["nombre_titular"]; ok && titVal != nil {
			nombreTit = fmt.Sprintf("%v", titVal)
		} else if titVal, ok := args["titular"]; ok && titVal != nil {
			nombreTit = fmt.Sprintf("%v", titVal)
		} else if pagVal, ok := args["pagador"]; ok && pagVal != nil {
			nombreTit = fmt.Sprintf("%v", pagVal)
		}

		reconRes, err := a.reconciliationUC.ReconcileCustomerOrder(ctx, orderID, input.DisambiguationOptions{
			OperationID: strings.TrimSpace(numeroOp),
			PayerName:   strings.TrimSpace(nombreTit),
		})
		if err != nil {
			return map[string]any{"error": fmt.Sprintf("Error verificando pago: %v", err)}
		}

		return map[string]any{
			"pedido_id": orderID,
			"status":    reconRes.Status,
			"pagado":    reconRes.Status == domain.ReconciliationStatusMatched,
			"mensaje":   reconRes.Message,
		}

	default:
		return map[string]any{"error": "Herramienta desconocida"}
	}
}

func (a *WhatsAppAgentUseCase) executeCrearPedido(
	ctx context.Context,
	customerPhone string,
	profileName string,
	args map[string]any,
) map[string]any {
	// 1. Check open cash register
	_, err := a.cashRepo.GetActive()
	if err != nil {
		return map[string]any{
			"error": "En este momento no hay una caja registradora abierta en el local para procesar nuevos pedidos.",
		}
	}

	// 2. Find or create customer by wa_id (never typed by user)
	customer, err := a.customerRepo.FindByWhatsapp(customerPhone)
	if err != nil || customer == nil {
		custName := profileName
		if strings.TrimSpace(custName) == "" {
			custName = "Cliente WhatsApp"
		}
		newCust := &domain.Customer{
			Name:     custName,
			Whatsapp: customerPhone,
		}
		if err := a.customerRepo.Save(newCust); err != nil {
			return map[string]any{"error": fmt.Sprintf("Error creando cliente: %v", err)}
		}
		customer = newCust
	}

	// 3. Process items array
	itemsRaw, ok := args["items"].([]any)
	if !ok || len(itemsRaw) == 0 {
		return map[string]any{"error": "El pedido debe contener al menos un producto."}
	}

	toppings, err := a.toppingRepo.GetAll()
	if err != nil {
		return map[string]any{"error": "Error consultando catálogo de sabores."}
	}

	var orderItems []domain.OrderItem
	for _, it := range itemsRaw {
		itemMap, ok := it.(map[string]any)
		if !ok {
			continue
		}
		sabor, _ := itemMap["sabor"].(string)
		formato, _ := itemMap["formato"].(string)
		var cant int
		switch v := itemMap["cantidad"].(type) {
		case float64:
			cant = int(v)
		case int:
			cant = v
		}
		if cant <= 0 {
			cant = 1
		}

		// Find topping by name
		var matchedTopping *domain.Topping
		for _, t := range toppings {
			if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(sabor)) ||
				strings.Contains(strings.ToLower(t.Name), strings.ToLower(sabor)) {
				matchedTopping = &t
				break
			}
		}
		if matchedTopping == nil {
			return map[string]any{"error": fmt.Sprintf("Sabor '%s' no disponible en nuestro menú.", sabor)}
		}

		isCombo := strings.EqualFold(formato, "combo")
		orderItems = append(orderItems, domain.OrderItem{
			ToppingID: matchedTopping.ID,
			Quantity:  cant,
			IsCombo:   isCombo,
		})
	}

	// 4. Delivery modality & cost
	modalidad, _ := args["modalidad_entrega"].(string)
	deliveryType := domain.DeliveryTypePickup
	var deliveryCost float64
	var addressIDPtr *uint

	if strings.EqualFold(modalidad, "envio") {
		deliveryType = domain.DeliveryTypeDelivery
		if addrVal, ok := args["address_id"]; ok {
			var addrID uint
			switch v := addrVal.(type) {
			case uint:
				addrID = v
			case int:
				addrID = uint(v)
			case int64:
				addrID = uint(v)
			case uint64:
				addrID = uint(v)
			case float64:
				addrID = uint(v)
			}
			addr, err := a.addressUseCase.GetAddressByID(addrID)
			if err == nil && addr != nil {
				addressIDPtr = &addrID
				quote, err := a.deliveryPort.QuoteDelivery(ctx, addr)
				if err == nil && quote != nil {
					deliveryCost = quote.EstimatedFee
				}
			}
		}
	}

	// 5. Create Order
	order := &domain.Order{
		CustomerID:    customer.ID,
		Status:        domain.OrderStatusPendingPayment,
		PaymentMethod: "TRANSFER_ALIAS",
		DeliveryType:  deliveryType,
		AddressID:     addressIDPtr,
		DeliveryCost:  deliveryCost,
		Items:         orderItems,
	}

	if err := a.orderUseCase.CreateOrder(order); err != nil {
		return map[string]any{"error": fmt.Sprintf("Error creando orden: %v", err)}
	}

	return map[string]any{
		"pedido_id":     order.ID,
		"status":        order.Status,
		"delivery_type": order.DeliveryType,
		"costo_envio":   order.DeliveryCost,
		"total_a_pagar": order.TotalAmount,
		"alias_pago":    "Caneroll.mp",
		"titular_pago":  "Ivonne Abreu",
		"mensaje":       "Pedido registrado con éxito en espera de transferencia al Alias Caneroll.mp.",
	}
}

// handleFallback handles conversations when the external LLM is offline or in mock mode
func (a *WhatsAppAgentUseCase) handleFallback(
	ctx context.Context,
	sessionCtx AgentSessionContext,
	userPrompt string,
	history []*genai.Content,
) (string, []*genai.Content, error) {
	lower := strings.ToLower(userPrompt)

	// 1. Payment verification
	if strings.Contains(lower, "pagu") || strings.Contains(lower, "pago") || strings.Contains(lower, "transfer") || strings.Contains(lower, "comprobante") {
		var pendingOrderID uint
		if sessionCtx.PendingOrderID != "" {
			if id, err := strconv.ParseUint(sessionCtx.PendingOrderID, 10, 32); err == nil {
				pendingOrderID = uint(id)
			}
		}
		if pendingOrderID == 0 && a.customerRepo != nil && a.cashRepo != nil {
			if cust, _ := a.customerRepo.FindByWhatsapp(sessionCtx.CustomerPhone); cust != nil {
				if activeReg, _ := a.cashRepo.GetActive(); activeReg != nil {
					if pending, _ := a.orderUseCase.GetPendingOrderByCustomer(activeReg.ID, cust.ID); pending != nil {
						pendingOrderID = pending.ID
					}
				}
			}
		}

		if pendingOrderID > 0 && a.reconciliationUC != nil {
			disambiguation := input.DisambiguationOptions{}
			for _, w := range strings.Fields(userPrompt) {
				clean := strings.Trim(w, ":,.-#")
				if strings.HasPrefix(strings.ToUpper(clean), "OP-") || (len(clean) >= 5 && isNumericString(clean)) {
					disambiguation.OperationID = clean
					break
				}
			}

			recon, err := a.reconciliationUC.ReconcileCustomerOrder(ctx, pendingOrderID, disambiguation)
			if err == nil && recon != nil {
				if recon.Status == domain.ReconciliationStatusMatched {
					return fmt.Sprintf("¡Muchas gracias! Ya verificamos tu transferencia en Mercado Pago por $%.2f y figura *Acreditada*. ¡Tu pedido #%d ya ingresó a cocina para preparación y despacho!\n\n🤎 Tus rolls salen recién horneados y listos para disfrutar.\nSi los vas a comer más tarde, te recomendamos calentarlos 5-10 segundos en microondas (el envase es apto 🔥) para que queden más suaves, esponjosos y el glaseado se vuelva aún más cremosito 😍\n¡Y si subís fotito a Instagram, etiquetanos en historias! 📸🤎", recon.MatchedTransaction.Amount, pendingOrderID), history, nil
				}
				return recon.Message, history, nil
			}
		}

		return "¡Muchas gracias! Ya verificamos tu pago en Mercado Pago y figura *Acreditado*. ¡Tu pedido ya ingresó a cocina para preparación y despacho!\n\n🤎 Tus rolls salen recién horneados y listos para disfrutar.\nSi los vas a comer más tarde, te recomendamos calentarlos 5-10 segundos en microondas (el envase es apto 🔥) para que queden más suaves, esponjosos y el glaseado se vuelva aún más cremosito 😍\n¡Y si subís fotito a Instagram, etiquetanos en historias! 📸🤎", history, nil
	}

	// 2. Order confirmation / placement
	if strings.Contains(lower, "confirm") || strings.Contains(lower, "pedir") || strings.Contains(lower, "quiero") {
		toppings, _ := a.toppingRepo.GetAll()
		var chosenTopping domain.Topping
		for _, t := range toppings {
			if t.IsActive {
				chosenTopping = t
				break
			}
		}

		modalidad := "retiro"
		if strings.Contains(lower, "envi") || strings.Contains(lower, "delivery") || strings.Contains(lower, "casa") {
			modalidad = "envio"
		}

		addrs, _ := a.addressUseCase.GetCustomerAddresses(sessionCtx.CustomerPhone)
		var addrIDVal any = 0
		if len(addrs) > 0 {
			addrIDVal = addrs[0].ID
		}

		args := map[string]any{
			"items": []any{
				map[string]any{
					"sabor":    chosenTopping.Name,
					"formato":  "combo",
					"cantidad": 1,
				},
			},
			"modalidad_entrega": modalidad,
			"address_id":        addrIDVal,
		}

		res := a.executeCrearPedido(ctx, sessionCtx.CustomerPhone, sessionCtx.CustomerName, args)
		if errMsg, hasErr := res["error"]; hasErr {
			return fmt.Sprintf("Ups: %v", errMsg), history, nil
		}

		total, _ := res["total_a_pagar"].(float64)
		id, _ := res["pedido_id"].(uint)

		return fmt.Sprintf("✨ ¡Gracias por tu pedido! ✨\n\n🧾 Pedido: Pedido #%d (1 Combo %s)\n🛵 Envío: %s\n\nTotal: *$%.2f*\n\n📲 Alias: Caneroll.mp\n👤 A nombre de: Ivonne Abreu\n\nUna vez realizada la transferencia, avisame 'ya pagué', 'ya transferí' o mandame el comprobante 🤎", id, chosenTopping.Name, modalidad, total), history, nil
	}

	// 3. Address inquiry / save
	if strings.Contains(lower, "direcci") || strings.Contains(lower, "calle") || strings.Contains(lower, "domicilio") {
		if strings.Contains(lower, "guard") || strings.Contains(lower, "av.") || strings.Contains(lower, "calle") {
			addr, _ := a.addressUseCase.SaveAddress(sessionCtx.CustomerPhone, "Casa", userPrompt, "")
			quote, _ := a.deliveryPort.QuoteDelivery(ctx, addr)
			return fmt.Sprintf("¡Excelente! Guardé tu dirección: *%s*. El costo de envío estimado a esa zona es de *$%.2f ARS*. ¿Confirmás tu pedido con esta dirección?", addr.StreetAndNumber, quote.EstimatedFee), history, nil
		}
		addrs, _ := a.addressUseCase.GetCustomerAddresses(sessionCtx.CustomerPhone)
		if len(addrs) == 0 {
			return "Hacemos envíos en Córdoba Capital. El costo depende de la zona 📍. Pasanos tu dirección o barrio para calcularlo 😊", history, nil
		}
		var sb strings.Builder
		sb.WriteString("Tus direcciones guardadas son:\n")
		for _, ad := range addrs {
			sb.WriteString(fmt.Sprintf("📍 *%s*: %s (%s)\n", ad.Alias, ad.StreetAndNumber, ad.ApartmentNotes))
		}
		sb.WriteString("¿A cuál te gustaría que enviemos tu pedido?")
		return sb.String(), history, nil
	}

	// 4. Flavor specific inquiry
	if strings.Contains(lower, "dulce de leche") {
		return "¡El rol de Dulce de Leche es una bomba! 🤎🍥 Masa artesanal súper suave y esponjosa recién horneada, con generoso dulce de leche argentino cremoso. Viene en formato individual ($4.000) o en combo x2 ($7.600). ¿Te gustaría pedir uno?", history, nil
	}
	if strings.Contains(lower, "pistacho") {
		return "¡El rol de Pistacho Premium es uno de los favoritos de CaneRolls! 🤎🍥 Masa suave recién horneada con crema y crocante de pistachos seleccionados. Sale $6.500 individual y $12.000 el combo x2. ¿Te tienta?", history, nil
	}
	if strings.Contains(lower, "oreo") {
		return "¡El rol de Oreo es riquísimo! 🤎🍥 Masa esponjosa recién salida del horno con glaseado y trocitos crocantes de galletitas Oreo. Sale $4.000 individual y $7.600 el combo x2. ¿Te gustaría pedir?", history, nil
	}
	if strings.Contains(lower, "avellana") || strings.Contains(lower, "nutella") {
		return "¡El rol de Avellana es espectacular! 🤎🍥 Masa suave con abundante crema de avellanas y chocolate. Sale $6.000 individual y $11.000 el combo x2. ¿Querés que te preparemos uno?", history, nil
	}
	if strings.Contains(lower, "clasico") || strings.Contains(lower, "clásico") {
		return "¡El Clásico es nuestro rol insignia! 🤎🍥 Masa suavecita con canela y nuestro frosting artesanal de queso crema que se derrite al calentarlo. Sale $4.000 individual y $7.000 el combo x2. ¿Te gustaría probarlo?", history, nil
	}

	// 5. Menu inquiry
	if strings.Contains(lower, "sabor") || strings.Contains(lower, "menu") || strings.Contains(lower, "menú") || strings.Contains(lower, "precio") || strings.Contains(lower, "carta") || strings.Contains(lower, "roles") || strings.Contains(lower, "rolls") {
		toppings, _ := a.toppingRepo.GetAll()
		var sb strings.Builder
		sb.WriteString("¡Te compartimos nuestro menú! 🤎🍥\nTodos nuestros rolls son caseros, suaves, esponjosos y recién horneados ✨\n📅 Tomamos pedidos de lunes a viernes.\n🚚 Entregamos viernes, sábados y domingos.\n⚠️ Trabajamos por pedidos y con cupos limitados por día para garantizar la mejor calidad 🍥✨\n\n")
		for _, t := range toppings {
			if t.IsActive {
				sb.WriteString(fmt.Sprintf("• *%s*: Individual $%.2f | Combo $%.2f\n", t.Name, t.SinglePrice, t.ComboPrice))
			}
		}
		sb.WriteString("\n¿Cuál te tienta probar hoy? 🤎")
		return sb.String(), history, nil
	}

	return "¡Hola! ¿Cómo andás? Te damos la bienvenida a CaneRolls 🤎🍥. Contame, ¿te gustaría conocer nuestro menú de hoy o querés armar un pedido?", history, nil
}

func isNumericString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

