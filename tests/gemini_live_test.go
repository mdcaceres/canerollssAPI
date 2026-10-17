package tests

import (
	"canerollss/adapters/output/datasource"
	"canerollss/adapters/output/delivery"
	"canerollss/adapters/output/payment"
	"canerollss/adapters/output/repository"
	"canerollss/app"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// TestLiveGeminiIntegration sends real prompts to Google Gemini in real time
// and prints the live response and function execution.
func TestLiveGeminiIntegration(t *testing.T) {
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" || apiKey == "mock_gemini_key" {
		t.Skip("Saltando prueba de Gemini en vivo: GEMINI_API_KEY aún no está guardada en .env")
	}

	// Use temporary SQLite database to avoid modifying Docker MySQL
	testDB := "canerollss_live_gemini_test.db"
	_ = os.Remove(testDB)
	defer os.Remove(testDB)

	os.Setenv("DB_DRIVER", "sqlite")
	os.Setenv("DB_NAME", testDB)
	os.Setenv("USE_MOCK_PROVIDERS", "true")

	// Initialize application and database
	_ = app.SetupApp()
	db := datasource.GetDB()
	cashRepo := repository.NewCashRegisterRepository(db)
	toppingRepo := repository.NewToppingRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	customerAddressRepo := repository.NewCustomerAddressRepository(db)

	cashUC := usecase.NewCashUseCase(cashRepo, orderRepo)
	orderUC := usecase.NewOrderUseCase(orderRepo, cashRepo, customerRepo, toppingRepo)
	addressUC := usecase.NewAddressUseCase(customerAddressRepo)

	deliveryPort := delivery.NewMockDeliveryProvider()
	paymentPort := payment.NewMockPaymentProvider()

	agentUC := usecase.NewWhatsAppAgentUseCase(orderUC, addressUC, customerRepo, toppingRepo, cashRepo, deliveryPort, paymentPort)

	// Ensure cash register is open so orders can be placed
	_, _ = cashUC.OpenRegister()

	customerPhone := "5491198765432"
	sessionCtx := agentUC.BuildSessionContext(customerPhone, "Matias")

	// -------------------------------------------------------------
	// TEST 1: Inquiry to Gemini
	// -------------------------------------------------------------
	userPrompt1 := "Hola Roli! Qué sabores de roles tienen y cuánto sale cada combo?"
	fmt.Println("\n=======================================================")
	fmt.Println(">> [PROMPT 1 A GEMINI (ROLI)]:", userPrompt1)
	fmt.Println("=======================================================")

	ctx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel1()

	reply1, history, err := agentUC.ProcessMessage(ctx1, sessionCtx, userPrompt1, nil)
	if err != nil {
		t.Fatalf("Error llamando a Gemini: %v", err)
	}

	fmt.Println("\n>> [RESPUESTA DE GEMINI EN VIVO]:")
	fmt.Println(reply1)
	fmt.Println("=======================================================")

	// -------------------------------------------------------------
	// TEST 2: Multi-turn Address & Order placement
	// -------------------------------------------------------------
	// Save customer address first
	_, _ = addressUC.SaveAddress(customerPhone, "Casa", "Av. Corrientes 1234 4B", "Timbre 4B")
	sessionCtx = agentUC.BuildSessionContext(customerPhone, "Matias")

	userPrompt2 := "Quiero pedir 1 combo de Clasico para entrega a mi Casa, confirmo mi pedido!"
	fmt.Println("\n=======================================================")
	fmt.Println(">> [PROMPT 2 A GEMINI (COTIZACIÓN Y RESUMEN)]:", userPrompt2)
	fmt.Println("=======================================================")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	reply2, history2, err := agentUC.ProcessMessage(ctx2, sessionCtx, userPrompt2, history)
	if err != nil {
		t.Fatalf("Error llamando a Gemini en turno 2: %v", err)
	}

	fmt.Println("\n>> [RESPUESTA DE GEMINI EN VIVO CON HERRAMIENTAS]:")
	fmt.Println(reply2)
	fmt.Println("=======================================================")

	// -------------------------------------------------------------
	// TEST 3: User says "Sí, confirmo!" -> Gemini calls crear_pedido_con_link_pago
	// -------------------------------------------------------------
	userPrompt3 := "Sí, confirmo! Dale para adelante."
	fmt.Println("\n=======================================================")
	fmt.Println(">> [PROMPT 3 A GEMINI (CONFIRMACIÓN FINAL CON LINK)]:", userPrompt3)
	fmt.Println("=======================================================")

	ctx3, cancel3 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel3()

	reply3, _, err := agentUC.ProcessMessage(ctx3, sessionCtx, userPrompt3, history2)
	if err != nil {
		t.Fatalf("Error llamando a Gemini en turno 3: %v", err)
	}

	fmt.Println("\n>> [RESPUESTA DE GEMINI CON LINK DE PAGO]:")
	fmt.Println(reply3)
	fmt.Println("=======================================================")

	// Verify order was created in DB by Gemini function call
	var createdOrder domain.Order
	if err := db.Last(&createdOrder).Error; err == nil {
		fmt.Printf("\n>> [VERIFICACIÓN EN BD] Pedido ID #%d creado en estado: %s | Costo Envio: $%.2f | Link: %s\n",
			createdOrder.ID, createdOrder.Status, createdOrder.DeliveryCost, createdOrder.PaymentLink)
	}
}
