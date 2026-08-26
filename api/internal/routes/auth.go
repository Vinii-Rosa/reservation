package routes

import "github.com/gin-gonic/gin"

func registerAuth(r *gin.Engine, h Handlers) {
	auth := r.Group("/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)

	protected := r.Group("/auth")
	protected.Use(h.AuthMiddleware.RequireAuth())
	protected.POST("/logout", h.Auth.Logout)
}
