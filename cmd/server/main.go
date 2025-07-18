package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := importer.ImportAllData(ctx, "https://github.com/yishak-cs/Neo4j_DB/data"); err != nil {
		log.Printf("Import failed: %v", err)
		os.Exit(1)
	}

	// Get import status
	_, err = importer.GetImportStatus(ctx)
	if err != nil {
		log.Printf("Failed to get import status: %v", err)
		os.Exit(1)
	}

	// Setup API routes
	apiHandler.SetupRoutes(router)

	// Handle React Router (SPA) - catch all other routes and serve index.html
	router.NoRoute(func(c *gin.Context) {
		// If the request is for API, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}
		// Otherwise serve the React app
		c.File("./web/static/index.html")
	})

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
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
