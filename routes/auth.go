package routes

import (
	"database/sql"
	handlers "finsec-backend/handlers/auth"
	"finsec-backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, db *sql.DB) {
	rg.Use(middleware.APIKeyMiddleware)
	rg.POST("/waitlist", handlers.Signup(db))
	rg.GET("/count", handlers.Counter(db))
}
