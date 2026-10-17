package datasource

import (
	"canerollss/core/domain"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func Connect() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	var conn *gorm.DB
	if os.Getenv("DB_DRIVER") == "sqlite" {
		dbPath := os.Getenv("DB_NAME")
		if dbPath == "" {
			dbPath = "canerollss_isolated.db"
		}
		c, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})
		if err != nil {
			log.Fatalf("Failed to connect to isolated SQLite database: %v", err)
		}
		conn = c
		log.Printf("Connected to isolated SQLite database: %s (production Docker DB untouched)", dbPath)
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		)

		c, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})

		if err != nil {
			log.Printf("Failed to connect to the database: %v", err)
			panic("could not connect to canerollss database")
		}
		conn = c
		log.Println("Database connection established successfully.")
	}

	DB = conn

	log.Println("Database connection established successfully.")

	err = conn.AutoMigrate(
		&domain.User{},
		&domain.Customer{},
		&domain.CustomerAddress{},
		&domain.Topping{},
		&domain.CashRegister{},
		&domain.Order{},
		&domain.OrderItem{},
		&domain.MonthlyClosing{},
		&domain.MonthlyToppingMetric{},
	)

	if err != nil {
		log.Printf("Could not migrate models: %v", err)
		return
	}

	log.Println("Database schema migration completed successfully.")

	seedAdminUser(conn)
	seedToppings(conn)
	seedCashRegister(conn)
}

func GetDB() *gorm.DB {
	return DB
}

func seedToppings(db *gorm.DB) {
	var count int64
	db.Model(&domain.Topping{}).Count(&count)

	if count == 0 {
		defaultToppings := []domain.Topping{
			{Name: "Clasico", SinglePrice: 4000, ComboPrice: 7000, IsActive: true},
			{Name: "Dulce de leche", SinglePrice: 4000, ComboPrice: 7600, IsActive: true},
			{Name: "Avellana", SinglePrice: 6000, ComboPrice: 11000, IsActive: true},
			{Name: "Oreo", SinglePrice: 4000, ComboPrice: 7600, IsActive: true},
			{Name: "Pistacho premium", SinglePrice: 6500, ComboPrice: 12000, IsActive: true},
		}

		for _, topping := range defaultToppings {
			db.Create(&topping)
		}
		log.Println("Default toppings seeded successfully.")
	}
}

func seedAdminUser(db *gorm.DB) {
	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	if adminUser == "" || adminPass == "" {
		adminUser = "admin"
		adminPass = "admin123"
	}

	var admin domain.User
	result := db.Where("role = ?", "ADMIN").First(&admin)

	if result.Error != nil {
		// Create new admin
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash seeder password: %v", err)
			return
		}

		newAdmin := domain.User{
			Username:     adminUser,
			PasswordHash: string(hash),
			Role:         "ADMIN",
			IsActive:     true,
		}

		if err := db.Create(&newAdmin).Error; err != nil {
			log.Printf("Failed to seed admin user: %v", err)
			return
		}

		log.Printf("Default admin user created successfully (username: %s)", adminUser)
	} else {
		// Update existing admin if password changed in .env
		err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(adminPass))
		if err != nil {
			hash, errHash := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
			if errHash == nil {
				admin.Username = adminUser
				admin.PasswordHash = string(hash)
				db.Save(&admin)
				log.Printf("Admin credentials updated from .env (username: %s)", adminUser)
			}
		} else if admin.Username != adminUser {
			admin.Username = adminUser
			db.Save(&admin)
			log.Printf("Admin username updated from .env (username: %s)", adminUser)
		}
	}
}

func seedCashRegister(db *gorm.DB) {
	if !strings.EqualFold(os.Getenv("AUTO_OPEN_REGISTER"), "true") {
		return
	}
	var count int64
	db.Model(&domain.CashRegister{}).Where("status = ?", domain.RegisterStatusOpen).Count(&count)
	if count == 0 {
		openReg := domain.CashRegister{
			OpenedAt: time.Now(),
			Status:   domain.RegisterStatusOpen,
		}
		if err := db.Create(&openReg).Error; err == nil {
			log.Println("Default open cash register initialized for development.")
		}
	}
}
