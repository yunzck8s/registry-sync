package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"registry-sync/internal/api/handlers"
	"registry-sync/internal/api/middleware"
	"registry-sync/internal/db/store"
	"registry-sync/internal/scheduler"
	ws "registry-sync/internal/websocket"
)

const version = "1.0.0"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func main() {
	// CLI flags
	var (
		port   = flag.String("port", "8080", "Server port")
		dbPath = flag.String("db", "registry-sync.db", "Database path")
		showVer = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("registry-sync server version %s\n", version)
		os.Exit(0)
	}

	// Initialize database
	log.Printf("Initializing database: %s", *dbPath)
	st, err := store.NewStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer st.Close()

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Initialize scheduler
	sched := scheduler.NewScheduler(st, hub)
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Middleware
	router.Use(middleware.CORS())

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "version": version})
		})

		// Registries
		registryHandler := handlers.NewRegistryHandler(st)
		v1.POST("/registries", registryHandler.CreateRegistry)
		v1.GET("/registries", registryHandler.ListRegistries)
		v1.GET("/registries/:id", registryHandler.GetRegistry)
		v1.PUT("/registries/:id", registryHandler.UpdateRegistry)
		v1.DELETE("/registries/:id", registryHandler.DeleteRegistry)
		v1.POST("/registries/:id/test", registryHandler.TestRegistry)
		v1.GET("/registries/:id/projects", registryHandler.ListProjects)
		v1.GET("/registries/:id/projects/:project/repositories", registryHandler.ListRepositories)

		// Tasks
		taskHandler := handlers.NewTaskHandler(st)
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.PUT("/tasks/:id", taskHandler.UpdateTask)
		v1.DELETE("/tasks/:id", taskHandler.DeleteTask)
		v1.POST("/tasks/:id/run", func(c *gin.Context) {
			// Parse task ID
			var taskID uint
			fmt.Sscanf(c.Param("id"), "%d", &taskID)

			// Execute task
			if err := sched.ExecuteTask(context.Background(), taskID); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "task started"})
		})
		v1.POST("/tasks/:id/stop", func(c *gin.Context) {
			// Parse task ID
			var taskID uint
			fmt.Sscanf(c.Param("id"), "%d", &taskID)

			// Cancel task
			if err := sched.CancelTask(taskID); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "task stopped"})
		})

		// Executions
		executionHandler := handlers.NewExecutionHandler(st)
		v1.GET("/executions", executionHandler.ListExecutions)
		v1.GET("/executions/:id", executionHandler.GetExecution)
		v1.GET("/executions/:id/logs", executionHandler.GetExecutionLogs)

		// Statistics
		v1.GET("/stats", executionHandler.GetStats)

		// Notifications
		notificationHandler := handlers.NewNotificationHandler(st)
		v1.POST("/notifications", notificationHandler.CreateNotificationChannel)
		v1.GET("/notifications", notificationHandler.ListNotificationChannels)
		v1.GET("/notifications/:id", notificationHandler.GetNotificationChannel)
		v1.PUT("/notifications/:id", notificationHandler.UpdateNotificationChannel)
		v1.DELETE("/notifications/:id", notificationHandler.DeleteNotificationChannel)
		v1.POST("/notifications/:id/test", notificationHandler.TestNotificationChannel)

		// WebSocket for real-time updates
		v1.GET("/ws", func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				log.Printf("WebSocket upgrade failed: %v", err)
				return
			}

			client := ws.NewClient(hub, conn)
			hub.Register(client)

			go client.WritePump()
			go client.ReadPump()
		})
	}

	// Serve static files (for frontend)
	router.StaticFS("/assets", http.Dir("./web/build/assets"))
	router.StaticFile("/favicon.ico", "./web/build/favicon.ico")

	// Serve index.html for root path
	router.GET("/", func(c *gin.Context) {
		c.File("./web/build/index.html")
	})

	// Serve index.html for all non-API routes (SPA fallback)
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't serve index.html for API or assets requests
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/assets/") {
			c.Status(404)
			return
		}

		// Serve index.html for client-side routing
		c.File("./web/build/index.html")
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: router,
	}

	// Start server in background
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s", *port)
		log.Printf("📊 API documentation: http://localhost:%s/api/v1/health", *port)
		log.Printf("🔌 WebSocket endpoint: ws://localhost:%s/api/v1/ws", *port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
