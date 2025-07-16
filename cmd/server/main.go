package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yishak-cs/Neo4j_DB/internal/database"
	"github.com/yishak-cs/Neo4j_DB/internal/handlers"
	"github.com/yishak-cs/Neo4j_DB/internal/services"
	"github.com/yishak-cs/Neo4j_DB/pkg/helper"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	// Initialize Neo4j client
	config := helper.LoadConfigFromEnv()
	neo4jClient, err := database.NewNeo4jClient(config)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := neo4jClient.Close(ctx); err != nil {
			log.Printf("Error closing Neo4j connection: %v", err)
		}
	}()

	// Initialize CSV importer
	importer := database.NewCSVImporter(neo4jClient)

	// Initialize services
	recommendationService := services.NewRecommendationService(neo4jClient)

	// Initialize API handlers
	apiHandler := handlers.NewAPIHandler(recommendationService)

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Setup routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "NeoRestro Recommendation Engine API",
			"status":  "running",
		})
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := neo4jClient.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": fmt.Sprintf("Database connection error: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Service is healthy",
		})
	})

	// Import data endpoint
	router.POST("/import", func(c *gin.Context) {
		baseURL := c.Query("baseURL")
		if baseURL == "" {
			// Use the server's own URL if not provided
			baseURL = fmt.Sprintf("http://%s", c.Request.Host)
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := importer.ImportAllData(ctx, baseURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": fmt.Sprintf("Import failed: %v", err),
			})
			return
		}

		// Get import status
		status, err := importer.GetImportStatus(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": fmt.Sprintf("Failed to get import status: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Data imported successfully",
			"stats":   status,
		})
	})

	// Setup API routes
	apiHandler.SetupRoutes(router)

	// Get port from environment or use default
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Gracefully shutdown with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
