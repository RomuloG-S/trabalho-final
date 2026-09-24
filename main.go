package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const version = "1.0.0"

func newRouter(store *Store) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Format(time.RFC3339), "version": version})
	})
	v1.POST("/salas", store.createRoom)
	v1.GET("/salas", store.listRooms)
	v1.GET("/salas/:id/grade", store.roomSchedule)
	v1.POST("/alunos", store.createStudent)
	v1.GET("/alunos", store.listStudents)
	v1.GET("/alunos/:id", store.getStudent)
	v1.POST("/turmas", store.createClass)
	v1.GET("/turmas", store.listClasses)
	v1.POST("/turmas/:id/alunos", store.enrollStudent)
	v1.GET("/turmas/:id/alunos", store.classStudents)
	v1.POST("/turmas/:id/alocar", store.allocateClass)
	return r
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := newRouter(NewStore()).Run(":" + port); err != nil {
		panic(err)
	}
}
