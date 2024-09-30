package godatabase

import (
	"log"
	"reflect"

	"gorm.io/gorm"
)

type database struct {
	*gorm.DB
	connection *Connection
}

func (x *database) Get() *gorm.DB {
	var reconnect bool
	if x.DB == nil {
		reconnect = true
	}

	if reflect.DeepEqual(x.DB, &gorm.DB{}) {
		reconnect = true
	}

	if !reconnect {
		if data, err := x.DB.DB(); err != nil {
			log.Println("Error get database connection", err.Error())
			reconnect = true
		} else {
			if err := data.Ping(); err != nil {
				log.Println("Error ping database connection", err.Error())
				reconnect = true
			}
		}
	}

	if reconnect {
		log.Println("Reconnect to LazisWallet database")
		if data, err := x.connection.reconnect(); err != nil {
			log.Println("Error reconnect to LazisWallet database", err.Error())
		} else {
			x.DB = data
		}
	}

	return x.DB
}
