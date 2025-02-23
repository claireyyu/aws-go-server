package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// We'll store a global *sql.DB for simplicity in a demo.
var db *sql.DB

func main() {
	// Read MySQL DSN from environment variable, e.g.:
	// DB_DSN = "user:pass@tcp(mydb.xxxx.us-west-2.rds.amazonaws.com:3306)/mydemodb"
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable not set")
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	// Test the DB connection quickly
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	db.Exec(`
	CREATE TABLE IF NOT EXISTS test_table (
		id INT AUTO_INCREMENT PRIMARY KEY,
		some_value INT
	) ENGINE=InnoDB;
	`)

	// Setup Gin engine
	r := gin.Default()

	// Health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// GET /count -> returns row count in "test_table"
	r.GET("/count", func(c *gin.Context) {
		var cnt int
		row := db.QueryRow("SELECT COUNT(*) FROM test_table")
		if err := row.Scan(&cnt); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"row_count": cnt})
	})

	// POST /insert -> inserts a row with some random value
	r.POST("/insert", func(c *gin.Context) {
		res, err := db.Exec("INSERT INTO test_table (some_value) VALUES (FLOOR(RAND()*1000))")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(200, gin.H{"message": "inserted", "row_id": id})
	})

	// Optionally, pass a port via environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s ...", port)
	r.Run(":" + port)
}
