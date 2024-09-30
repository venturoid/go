package godatabase

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Configuration struct {
	ParseTime bool
	Location  string
	Charset   string
	Retries   int
}

type Connection struct {
	// database connection information
	Host     string
	Port     int
	Username string
	Password string
	Database string

	// database connection configuration
	Configuration
}

func NewConnection(config Connection) (*database, error) {
	// set default values
	if config.Retries == 0 {
		config.Retries = 3
	}

	// setup connection configuration
	connection := &Connection{
		Host:     config.Host,
		Port:     config.Port,
		Username: config.Username,
		Password: config.Password,
		Database: config.Database,
		Configuration: Configuration{
			Retries:   config.Retries,
			ParseTime: config.ParseTime,
			Location:  config.Location,
			Charset:   config.Charset,
		},
	}

	// connect to database
	var db *gorm.DB
	if data, err := connection.connect(); err != nil {
		log.Println("error on creating connection gorm database", err.Error())
	} else {
		db = data
		config.Retries = connection.Retries
	}

	if db == nil {
		if data, err := connection.reconnect(); err != nil {
			log.Println("error on reconnecting to database", err.Error())
			return nil, err
		} else {
			db = data
		}
	}

	return &database{
		DB:         db,
		connection: &config,
	}, nil
}

func (c *Connection) connect() (*gorm.DB, error) {

	// build DNS string
	dns := buildDNS(*c)

	log.Println(dns)

	// connect to database
	db, err := sql.Open("mysql", dns)
	if err != nil {
		log.Printf("error on creating connection sql database %v", err.Error())
		return nil, err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(time.Second * 10)

	// assign database connection
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		log.Println("error disini")
		return nil, err
	}

	// setup logger
	newLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		})
	gormDB.Session(&gorm.Session{Logger: newLogger})

	log.Println("database connection successfully")
	return gormDB, nil
}

func (c *Connection) reconnect() (*gorm.DB, error) {
	retry := c.Retries
	// reconnect to database
	if retry == 0 {
		return nil, fmt.Errorf("failed to reconnect to database")
	}
	var db *gorm.DB
	for retry > 0 {
		if data, err := c.connect(); err != nil {
			select {
			case <-time.After(5 * time.Second):
				log.Println("reconnect")
				retry--
			}
		} else {
			db = data
			break
		}
	}

	return db, nil
}
