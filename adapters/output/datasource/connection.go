package datasource

import (
	"canerollss/core/domain"
	"fmt"
	"log"
	"os"

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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})

	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
		panic("could not connect to canerollss database")
	}

	DB = conn

	log.Println("Database connection established successfully.")

	err = conn.AutoMigrate(
		&domain.User{},
		&domain.Customer{},
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
}

func GetDB() *gorm.DB {
	return DB
}

func seedAdminUser(db *gorm.DB) {
	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	if adminUser == "" || adminPass == "" {
		adminUser = "admin"
		adminPass = "admin123"
	}

	var count int64
	db.Model(&domain.User{}).Where("role = ?", "ADMIN").Count(&count)

	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash seeder password: %v", err)
			return
		}

		admin := domain.User{
			Username:     adminUser,
			PasswordHash: string(hash),
			Role:         "ADMIN",
			IsActive:     true,
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Printf("Failed to seed admin user: %v", err)
			return
		}

		log.Printf("Default admin user created successfully (username: %s)", adminUser)
	}
}
