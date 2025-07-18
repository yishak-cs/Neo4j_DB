package services

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/yishak-cs/Neo4j_DB/internal/database"
	"github.com/yishak-cs/Neo4j_DB/internal/models"
)

// RecommendationService handles all recommendation logic
type RecommendationService struct {
	client *database.Neo4jClient
}

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(client *database.Neo4jClient) *RecommendationService {
	return &RecommendationService{
		client: client,
	}
}

// GetUserFrequentItems answers: "What does a user generally order most frequently?"
func (s *RecommendationService) GetUserFrequentItems(ctx context.Context, userID int) ([]models.Recommendation, error) {
	query := `
		MATCH (u:User {db_id: $userId})-[ho:HAS_ORDERED]->(i:Item)
		RETURN i.db_id AS item_id, 
			   i.name AS name, 
			   i.price AS price, 
			   i.category AS category, 
			   ho.times AS times
		ORDER BY ho.times DESC
	`

	params := map[string]interface{}{
		"userId": userID,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user frequent items: %w", err)
	}

	var recommendations []models.Recommendation
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["item_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		times := int(result["times"].(int64))

		recommendations = append(recommendations, models.Recommendation{
			Item:        item,
			Score:       float64(times),
			Explanation: fmt.Sprintf("You've ordered this %d times", times),
			Strategy:    "UserFrequency",
		})
	}

	return recommendations, nil
}

// GetUserCoOrderedItems answers: "Once item X is in cart, what did THIS user previously order with X?"
func (s *RecommendationService) GetUserCoOrderedItems(ctx context.Context, userID int, itemInCartID int) ([]models.Recommendation, error) {
	query := `
		MATCH (u:User {db_id: $userId})-[:HAS_MADE]->(o:Order)-[:HAS_ITEM]->(target:Item {db_id: $itemInCart})
		MATCH (o)-[:HAS_ITEM]->(coItem:Item)
		WHERE coItem.db_id <> $itemInCart
		WITH coItem, count(o) as coOccurrences
		RETURN coItem.db_id AS item_id, 
			   coItem.name AS name, 
			   coItem.price AS price, 
			   coItem.category AS category, 
			   coOccurrences
		ORDER BY coOccurrences DESC
	`

	params := map[string]interface{}{
		"userId":     userID,
		"itemInCart": itemInCartID,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user co-ordered items: %w", err)
	}

	var recommendations []models.Recommendation
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["item_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		coOccurrences := int(result["coOccurrences"].(int64))

		recommendations = append(recommendations, models.Recommendation{
			Item:        item,
			Score:       float64(coOccurrences),
			Explanation: fmt.Sprintf("You've ordered this %d times with %d", coOccurrences, itemInCartID),
			Strategy:    "UserCoOrders",
		})
	}

	return recommendations, nil
}

// GetGlobalCoOrderedItems answers: "Once item X is in cart, what items are frequently ordered with X across ALL users?"
func (s *RecommendationService) GetGlobalCoOrderedItems(ctx context.Context, itemInCartID int) ([]models.Recommendation, error) {
	query := `
		MATCH (target:Item {db_id: $itemInCart})-[oaw:ORDERED_ALONG_WITH]->(coItem:Item)
		RETURN coItem.db_id AS item_id, 
			   coItem.name AS name, 
			   coItem.price AS price, 
			   coItem.category AS category, 
			   oaw.times AS times
		ORDER BY oaw.times DESC
	`

	params := map[string]interface{}{
		"itemInCart": itemInCartID,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get global co-ordered items: %w", err)
	}

	var recommendations []models.Recommendation
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["item_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		times := int(result["times"].(int64))

		recommendations = append(recommendations, models.Recommendation{
			Item:        item,
			Score:       float64(times),
			Explanation: fmt.Sprintf("Customers who ordered item %d also ordered this %d times", itemInCartID, times),
			Strategy:    "GlobalCoOrders",
		})
	}

	return recommendations, nil
}

// GetTimeBasedTrendingItems gets items trending in the last N days
func (s *RecommendationService) GetTimeBasedTrendingItems(ctx context.Context, days int) ([]models.Recommendation, error) {
	query := `
		MATCH (o:Order)-[:HAS_ITEM]->(i:Item)
		WHERE date(o.created_at) > date() - duration({days: $days})
		WITH i, count(o) as recent_orders
		RETURN i.db_id AS item_id, 
        i.name AS name, 
        i.price AS price, 
        i.category AS category, 
        recent_orders
		ORDER BY recent_orders DESC
	`

	params := map[string]interface{}{
		"days": days,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending items: %w", err)
	}

	var recommendations []models.Recommendation
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["item_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		recentOrders := int(result["recent_orders"].(int64))

		recommendations = append(recommendations, models.Recommendation{
			Item:        item,
			Score:       float64(recentOrders),
			Explanation: fmt.Sprintf("Ordered %d times in the last %d days", recentOrders, days),
			Strategy:    "TimeBasedTrend",
		})
	}

	return recommendations, nil
}

// HybridRecommendation combines all recommendation strategies with weights
func (s *RecommendationService) HybridRecommendation(ctx context.Context, userID int, itemInCartID *int, weights models.HybridWeights) ([]models.Recommendation, error) {
	log.Printf("Generating hybrid recommendations for user %d with item in cart %v", userID, itemInCartID)

	// Track all items and their scores
	itemScores := make(map[int]float64)
	itemDetails := make(map[int]models.Item)
	strategyContributions := make(map[int]map[string]float64)

	// 1. Get user frequency recommendations
	userFreqRecs, err := s.GetUserFrequentItems(ctx, userID)
	if err != nil {
		log.Printf("Warning: Failed to get user frequency recommendations: %v", err)
	} else {
		for _, rec := range userFreqRecs {
			itemID := rec.Item.DbID
			score := rec.Score * weights.UserFrequency

			itemScores[itemID] = itemScores[itemID] + score
			itemDetails[itemID] = rec.Item

			if strategyContributions[itemID] == nil {
				strategyContributions[itemID] = make(map[string]float64)
			}
			strategyContributions[itemID]["UserFrequency"] = score
		}
	}

	// 2. If item in cart, get co-ordered items
	if itemInCartID != nil {
		// User co-orders
		userCoRecs, err := s.GetUserCoOrderedItems(ctx, userID, *itemInCartID)
		if err != nil {
			log.Printf("Warning: Failed to get user co-ordered recommendations: %v", err)
		} else {
			for _, rec := range userCoRecs {
				itemID := rec.Item.DbID
				score := rec.Score * weights.UserCoOrders

				itemScores[itemID] = itemScores[itemID] + score
				itemDetails[itemID] = rec.Item

				if strategyContributions[itemID] == nil {
					strategyContributions[itemID] = make(map[string]float64)
				}
				strategyContributions[itemID]["UserCoOrders"] = score
			}
		}

		// Global co-orders
		globalCoRecs, err := s.GetGlobalCoOrderedItems(ctx, *itemInCartID)
		if err != nil {
			log.Printf("Warning: Failed to get global co-ordered recommendations: %v", err)
		} else {
			for _, rec := range globalCoRecs {
				itemID := rec.Item.DbID
				score := rec.Score * weights.GlobalCoOrders

				itemScores[itemID] = itemScores[itemID] + score
				itemDetails[itemID] = rec.Item

				if strategyContributions[itemID] == nil {
					strategyContributions[itemID] = make(map[string]float64)
				}
				strategyContributions[itemID]["GlobalCoOrders"] = score
			}
		}
	}

	// 3. Get trending items
	trendRecs, err := s.GetTimeBasedTrendingItems(ctx, 7) // Last 7 days
	if err != nil {
		log.Printf("Warning: Failed to get trending recommendations: %v", err)
	} else {
		for _, rec := range trendRecs {
			itemID := rec.Item.DbID
			score := rec.Score * weights.TimeBasedTrend

			itemScores[itemID] = itemScores[itemID] + score
			itemDetails[itemID] = rec.Item

			if strategyContributions[itemID] == nil {
				strategyContributions[itemID] = make(map[string]float64)
			}
			strategyContributions[itemID]["TimeBasedTrend"] = score
		}
	}

	// Filter out the item in cart if it exists
	if itemInCartID != nil {
		delete(itemScores, *itemInCartID)
		delete(itemDetails, *itemInCartID)
		delete(strategyContributions, *itemInCartID)
	}

	// Convert to slice for sorting
	var recommendations []models.Recommendation
	for itemID, totalScore := range itemScores {
		// Find the strategy that contributed most to this recommendation
		var topStrategy string
		var topContribution float64

		for strategy, contribution := range strategyContributions[itemID] {
			if contribution > topContribution {
				topStrategy = strategy
				topContribution = contribution
			}
		}

		// Generate explanation based on top strategy
		var explanation string
		switch topStrategy {
		case "UserFrequency":
			explanation = "Recommended because you frequently order this"
		case "UserCoOrders":
			explanation = fmt.Sprintf("You often order this with item %d", *itemInCartID)
		case "GlobalCoOrders":
			explanation = fmt.Sprintf("Customers who order item %d also order this", *itemInCartID)
		case "TimeBasedTrend":
			explanation = "This item is trending right now"
		default:
			explanation = "Recommended based on your preferences"
		}

		recommendations = append(recommendations, models.Recommendation{
			Item:        itemDetails[itemID],
			Score:       totalScore,
			Explanation: explanation,
			Strategy:    topStrategy,
		})
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations, nil
}

// GetDefaultWeights returns the default weights for hybrid recommendations
func (s *RecommendationService) GetDefaultWeights() models.HybridWeights {
	return models.HybridWeights{
		UserFrequency:  0.4,
		UserCoOrders:   0.3,
		GlobalCoOrders: 0.2,
		TimeBasedTrend: 0.1,
	}
}

// GetWeightsForNewUser returns weights optimized for new users
func (s *RecommendationService) GetWeightsForNewUser() models.HybridWeights {
	return models.HybridWeights{
		UserFrequency:  0.1,
		UserCoOrders:   0.1,
		GlobalCoOrders: 0.5,
		TimeBasedTrend: 0.3,
	}
}

// GetWeightsForExperiencedUser returns weights optimized for experienced users
func (s *RecommendationService) GetWeightsForExperiencedUser() models.HybridWeights {
	return models.HybridWeights{
		UserFrequency:  0.5,
		UserCoOrders:   0.3,
		GlobalCoOrders: 0.1,
		TimeBasedTrend: 0.1,
	}
}

// IsNewUser determines if a user is new based on order history
func (s *RecommendationService) IsNewUser(ctx context.Context, userID int) (bool, error) {
	query := `
		MATCH (u:User {db_id: $userId})-[:HAS_MADE]->(o:Order)
		RETURN count(o) as order_count
	`

	params := map[string]interface{}{
		"userId": userID,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return true, fmt.Errorf("failed to check user status: %w", err)
	}

	if len(results) == 0 {
		return true, nil
	}

	orderCount := int(results[0]["order_count"].(int64))
	return orderCount < 3, nil // Consider users with less than 3 orders as new
}

// GetAllItems retrieves all menu items from the database
func (s *RecommendationService) GetAllItems(ctx context.Context) ([]models.Item, error) {
	query := `
		MATCH (i:Item)
		RETURN i.db_id AS db_id, 
			   i.name AS name, 
			   i.price AS price, 
			   i.category AS category,
			   i.description AS description
		ORDER BY i.category, i.name
	`

	results, err := s.client.ExecuteRead(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}

	var items []models.Item
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["db_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		// Handle optional description field
		if desc, ok := result["description"]; ok && desc != nil {
			if descStr, ok := desc.(string); ok {
				item.Description = descStr
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// GetItemsByCategory retrieves menu items by category
func (s *RecommendationService) GetItemsByCategory(ctx context.Context, category string) ([]models.Item, error) {
	query := `
		MATCH (i:Item {category: $category})
		RETURN i.db_id AS db_id, 
			   i.name AS name, 
			   i.price AS price, 
			   i.category AS category,
			   i.description AS description
		ORDER BY i.name
	`

	params := map[string]interface{}{
		"category": category,
	}

	results, err := s.client.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by category: %w", err)
	}

	var items []models.Item
	for _, result := range results {
		item := models.Item{
			DbID:     int(result["db_id"].(int64)),
			Name:     result["name"].(string),
			Price:    result["price"].(float64),
			Category: result["category"].(string),
		}

		// Handle optional description field
		if desc, ok := result["description"]; ok && desc != nil {
			if descStr, ok := desc.(string); ok {
				item.Description = descStr
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// GetAllUsers retrieves all users from the database
func (s *RecommendationService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	query := `
		MATCH (u:User)
		RETURN u.db_id AS db_id, 
			   u.name AS name, 
			   u.email AS email,
			   u.created_at AS created_at
		ORDER BY u.name
	`

	results, err := s.client.ExecuteRead(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	var users []models.User
	for _, result := range results {
		user := models.User{
			DbID:  int(result["db_id"].(int64)),
			Name:  result["name"].(string),
			Email: result["email"].(string),
		}

		// Handle created_at if present
		if createdAt, ok := result["created_at"]; ok && createdAt != nil {
			// Neo4j datetime handling would go here if needed
			// For now, we'll skip the datetime conversion
		}

		users = append(users, user)
	}

	return users, nil
}
