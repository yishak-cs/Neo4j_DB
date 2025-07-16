package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yishak-cs/Neo4j_DB/internal/models"
	"github.com/yishak-cs/Neo4j_DB/internal/services"
)

// APIHandler handles all API requests
type APIHandler struct {
	recommendationService *services.RecommendationService
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(recommendationService *services.RecommendationService) *APIHandler {
	return &APIHandler{
		recommendationService: recommendationService,
	}
}

// SetupRoutes configures all API routes
func (h *APIHandler) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/recommendations/user-frequent/:userId", h.GetUserFrequentItems)
		api.GET("/recommendations/user-co-orders/:userId/:itemId", h.GetUserCoOrderedItems)
		api.GET("/recommendations/global-co-orders/:itemId", h.GetGlobalCoOrderedItems)
		api.GET("/recommendations/trending", h.GetTrendingItems)
		api.GET("/recommendations/hybrid/:userId", h.GetHybridRecommendations)
	}
}

// GetUserFrequentItems handles requests for a user's most frequently ordered items
func (h *APIHandler) GetUserFrequentItems(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	recommendations, err := h.recommendationService.GetUserFrequentItems(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Error getting user frequent items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":         userID,
		"recommendations": recommendations,
		"strategy":        "UserFrequency",
		"description":     "Items you order most frequently",
	})
}

// GetUserCoOrderedItems handles requests for items a user frequently orders with a specific item
func (h *APIHandler) GetUserCoOrderedItems(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	itemID, err := strconv.Atoi(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	recommendations, err := h.recommendationService.GetUserCoOrderedItems(c.Request.Context(), userID, itemID)
	if err != nil {
		log.Printf("Error getting user co-ordered items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":         userID,
		"item_id":         itemID,
		"recommendations": recommendations,
		"strategy":        "UserCoOrders",
		"description":     "Items you frequently order with this item",
	})
}

// GetGlobalCoOrderedItems handles requests for items frequently ordered with a specific item by all users
func (h *APIHandler) GetGlobalCoOrderedItems(c *gin.Context) {
	itemID, err := strconv.Atoi(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	recommendations, err := h.recommendationService.GetGlobalCoOrderedItems(c.Request.Context(), itemID)
	if err != nil {
		log.Printf("Error getting global co-ordered items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item_id":         itemID,
		"recommendations": recommendations,
		"strategy":        "GlobalCoOrders",
		"description":     "Items frequently ordered with this item by all customers",
	})
}

// GetTrendingItems handles requests for currently trending items
func (h *APIHandler) GetTrendingItems(c *gin.Context) {
	days := 7 // Default to 7 days
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	recommendations, err := h.recommendationService.GetTimeBasedTrendingItems(c.Request.Context(), days)
	if err != nil {
		log.Printf("Error getting trending items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"days":            days,
		"recommendations": recommendations,
		"strategy":        "TimeBasedTrend",
		"description":     "Currently trending items",
	})
}

// GetHybridRecommendations handles requests for hybrid recommendations
func (h *APIHandler) GetHybridRecommendations(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Optional item in cart
	var itemInCartID *int
	if itemIDParam := c.Query("itemInCart"); itemIDParam != "" {
		if parsedItemID, err := strconv.Atoi(itemIDParam); err == nil {
			itemInCartID = &parsedItemID
		}
	}

	// Determine appropriate weights based on user experience
	var weights models.HybridWeights
	isNewUser, err := h.recommendationService.IsNewUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Error checking user status: %v", err)
		weights = h.recommendationService.GetDefaultWeights()
	} else if isNewUser {
		weights = h.recommendationService.GetWeightsForNewUser()
	} else {
		weights = h.recommendationService.GetWeightsForExperiencedUser()
	}

	// Allow weight customization via query params
	if userFreq := c.Query("userFreq"); userFreq != "" {
		if parsedWeight, err := strconv.ParseFloat(userFreq, 64); err == nil {
			weights.UserFrequency = parsedWeight
		}
	}
	if userCoOrders := c.Query("userCoOrders"); userCoOrders != "" {
		if parsedWeight, err := strconv.ParseFloat(userCoOrders, 64); err == nil {
			weights.UserCoOrders = parsedWeight
		}
	}
	if globalCoOrders := c.Query("globalCoOrders"); globalCoOrders != "" {
		if parsedWeight, err := strconv.ParseFloat(globalCoOrders, 64); err == nil {
			weights.GlobalCoOrders = parsedWeight
		}
	}
	if timeTrend := c.Query("timeTrend"); timeTrend != "" {
		if parsedWeight, err := strconv.ParseFloat(timeTrend, 64); err == nil {
			weights.TimeBasedTrend = parsedWeight
		}
	}

	recommendations, err := h.recommendationService.HybridRecommendation(
		c.Request.Context(),
		userID,
		itemInCartID,
		weights,
	)
	if err != nil {
		log.Printf("Error getting hybrid recommendations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	// Limit results to top 10
	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":         userID,
		"item_in_cart":    itemInCartID,
		"weights":         weights,
		"recommendations": recommendations,
		"strategy":        "Hybrid",
		"description":     "Personalized recommendations based on multiple factors",
	})
}
