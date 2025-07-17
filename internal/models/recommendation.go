package models

import "time"

// User represents a restaurant customer
type User struct {
	DbID      int       `json:"db_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Item represents a menu item
type Item struct {
	DbID        int     `json:"db_id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Description string  `json:"description,omitempty"`
}

// Order represents a customer order
type Order struct {
	DbID        int       `json:"db_id"`
	CreatedAt   time.Time `json:"created_at"`
	TotalAmount float64   `json:"total_amount"`
}

// HybridWeights represents the weights for different recommendation strategies
type HybridWeights struct {
	UserFrequency  float64 `json:"user_frequency"`
	UserCoOrders   float64 `json:"user_co_orders"`
	GlobalCoOrders float64 `json:"global_co_orders"`
	TimeBasedTrend float64 `json:"time_based_trend"`
}

// Recommendation represents a recommended item with its score and explanation
type Recommendation struct {
	Item        Item    `json:"item"`
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation"`
	Strategy    string  `json:"strategy"`
}

// OrderItem represents the relationship between an order and an item
type OrderItem struct {
	OrderID  int `json:"order_id"`
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// CoOccurrence represents how often items are ordered together
type CoOccurrence struct {
	ItemID        int       `json:"item_id"`
	CoItemID      int       `json:"co_item_id"`
	Times         int       `json:"times"`
	Correlation   float64   `json:"correlation"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}
