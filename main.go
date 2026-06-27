package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"finsec-backend/middleware"
	"finsec-backend/routes"
	"finsec-backend/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/joho/godotenv"
)

var rdb *redis.Client
var db *sql.DB

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"), // e.g. localhost:6379
		PoolSize: 200,                     // tune upward as VU count grows
	})
	log.Println("Redis client initialised")
}

func initDB() {
	var err error
	db, err = sql.Open("pgx", os.Getenv("DATABASE"))
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
	log.Println("DB connected")

	// DB settings
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found", err)
	}

	utils.Load()
	utils.InitResend()
	initDB()
	initRedis()

	store := middleware.NewRedisStore(rdb) // 👈 replaces the libredis block

	allowedOrigins := []string{}
	if dev := os.Getenv("DEV_SERVER"); dev != "" {
		allowedOrigins = append(allowedOrigins, dev)
	}
	if prod := os.Getenv("FRONTEND_PROD"); prod != "" {
		allowedOrigins = append(allowedOrigins, prod)
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173"}
	}

	router := gin.Default()

	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Upgrade", "Connection"},
		AllowCredentials: true,
		AllowWildcard:    true,
	}))
	router.Use(middleware.WrapGin(middleware.GlobalRateLimiter(store))) // router using middleware for ratelimit

	api := router.Group("/api")
	api.Use(middleware.WrapGin(middleware.IPRateLimiter(store))) // router using middleware to rate limit IPs

	routes.RegisterAuthRoutes(api.Group(""), db)

	router.Run(":9000")
}
