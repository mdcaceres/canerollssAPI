package datasource

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	conn, err := gorm.Open(mysql.Open("root:admin@/canerollss_db?parseTime=true"), &gorm.Config{})

	if err != nil {
		panic("could not connect to canerollss data base")
	}

	DB = conn

	err = conn.AutoMigrate()
	if err != nil {
		fmt.Println("could not migrate models")
		fmt.Println(err)
		return
	}
}

func GetDB() *gorm.DB {
	return DB

}
