package database

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
)

// CSVImporter handles importing CSV data into Neo4j
type CSVImporter struct {
	client *Neo4jClient
}

// NewCSVImporter creates a new CSV importer
func NewCSVImporter(client *Neo4jClient) *CSVImporter {
	return &CSVImporter{client: client}
}

// ImportAllData imports all CSV files in the correct order
func (i *CSVImporter) ImportAllData(ctx context.Context, baseURL string) error {
	log.Println("Starting CSV import process...")

	// Step 1: Clear existing data (optional - for development)
	if err := i.clearDatabase(ctx); err != nil {
		return fmt.Errorf("failed to clear database: %w", err)
	}

	// Step 2: Import in dependency order
	steps := []struct {
		name string
		fn   func(context.Context, string) error
	}{
		{"users", i.ImportUsers},
		{"items", i.ImportItems},
		{"orders", i.ImportOrders},
		{"order_items", i.ImportOrderItems},
		{"build_relationships", i.BuildRelationships},
	}

	for _, step := range steps {
		log.Printf("Importing %s...", step.name)
		if err := step.fn(ctx, baseURL); err != nil {
			return fmt.Errorf("failed to import %s: %w", step.name, err)
		}
		log.Printf("Successfully imported %s", step.name)
	}

	log.Println("CSV import process completed successfully")
	return nil
}

// ImportUsers imports users from CSV
func (i *CSVImporter) ImportUsers(ctx context.Context, baseURL string) error {
	filePath := "data/users.csv"
	records, err := readCsvFile(filePath)
	if err != nil {
		return err
	}

	// The first record is the header
	header := records[0]
	dataRows := records[1:]

	query := `
		UNWIND $rows as row
		MERGE (u:User {
			db_id: toInteger(row.user_id),
			name: row.name,
			email: row.email,
			created_at: datetime(row.created_at)
		})
		RETURN count(u) as imported_users
	`

	// Convert rows to a slice of maps
	var userList []map[string]interface{}
	for _, record := range dataRows {
		user := make(map[string]interface{})
		for j, value := range record {
			user[header[j]] = strings.TrimSpace(value)
		}
		userList = append(userList, user)
	}

	params := map[string]interface{}{
		"rows": userList,
	}

	return i.client.ExecuteWrite(ctx, query, params)
}

// ImportItems imports menu items from CSV
func (i *CSVImporter) ImportItems(ctx context.Context, baseURL string) error {
	filePath := "data/items.csv"
	records, err := readCsvFile(filePath)
	if err != nil {
		return err
	}

	header := records[0]
	dataRows := records[1:]

	query := `
		UNWIND $rows as row
		MERGE (i:Item {
			db_id: toInteger(row.item_id),
			name: row.name,
			price: toFloat(row.price),
			category: row.category,
			description: row.description
		})
		RETURN count(i) as imported_items
	`

	var itemList []map[string]interface{}
	for _, record := range dataRows {
		item := make(map[string]interface{})
		for j, value := range record {
			item[header[j]] = strings.TrimSpace(value)
		}
		itemList = append(itemList, item)
	}

	params := map[string]interface{}{
		"rows": itemList,
	}

	return i.client.ExecuteWrite(ctx, query, params)
}

// ImportOrders imports orders from CSV
func (i *CSVImporter) ImportOrders(ctx context.Context, baseURL string) error {
	filePath := "data/orders.csv"
	records, err := readCsvFile(filePath)
	if err != nil {
		return err
	}

	header := records[0]
	dataRows := records[1:]

	query := `
		UNWIND $rows as row
		MATCH (u:User {db_id: toInteger(row.user_id)})
		MERGE (o:Order {
			db_id: toInteger(row.order_id),
			created_at: datetime(row.created_at),
			total_amount: toFloat(row.total_amount)
		})
		MERGE (u)-[:HAS_MADE]->(o)
		RETURN count(o) as imported_orders
	`

	var orderList []map[string]interface{}
	for _, record := range dataRows {
		order := make(map[string]interface{})
		for j, value := range record {
			order[header[j]] = strings.TrimSpace(value)
		}
		orderList = append(orderList, order)
	}

	params := map[string]interface{}{
		"rows": orderList,
	}

	return i.client.ExecuteWrite(ctx, query, params)
}

// ImportOrderItems imports order-item relationships from CSV
func (i *CSVImporter) ImportOrderItems(ctx context.Context, baseURL string) error {
	filePath := "data/order_items.csv"
	records, err := readCsvFile(filePath)
	if err != nil {
		return err
	}

	header := records[0]
	dataRows := records[1:]

	query := `
		UNWIND $rows as row
		MATCH (o:Order {db_id: toInteger(row.order_id)})
		MATCH (i:Item {db_id: toInteger(row.item_id)})
		MERGE (o)-[:HAS_ITEM {
			quantity: toInteger(row.quantity)
		}]->(i)
		RETURN count(*) as imported_order_items
	`

	var orderItemList []map[string]interface{}
	for _, record := range dataRows {
		orderItem := make(map[string]interface{})
		for j, value := range record {
			orderItem[header[j]] = strings.TrimSpace(value)
		}
		orderItemList = append(orderItemList, orderItem)
	}

	params := map[string]interface{}{
		"rows": orderItemList,
	}

	return i.client.ExecuteWrite(ctx, query, params)
}

// BuildRelationships builds the derived relationships for recommendations
func (i *CSVImporter) BuildRelationships(ctx context.Context, baseURL string) error {
	log.Println("Building HAS_ORDERED relationships...")
	if err := i.buildHasOrderedRelationships(ctx); err != nil {
		return fmt.Errorf("failed to build HAS_ORDERED relationships: %w", err)
	}

	log.Println("Building ORDERED_ALONG_WITH relationships...")
	if err := i.buildOrderedAlongWithRelationships(ctx); err != nil {
		return fmt.Errorf("failed to build ORDERED_ALONG_WITH relationships: %w", err)
	}

	return nil
}

// buildHasOrderedRelationships creates aggregated user-item relationships
func (i *CSVImporter) buildHasOrderedRelationships(ctx context.Context) error {
	query := `
		MATCH (u:User)-[:HAS_MADE]->(o:Order)-[hi:HAS_ITEM]->(i:Item)
		WITH u, i, sum(hi.quantity) as total_quantity
		MERGE (u)-[ho:HAS_ORDERED]->(i)
		SET ho.times = total_quantity
		RETURN count(ho) as created_relationships
	`

	// Changed from ExecuteRead to ExecuteWrite
	err := i.client.ExecuteWrite(ctx, query, nil)
	if err != nil {
		return err
	}

	log.Println("Built HAS_ORDERED relationships successfully")
	return nil
}

// buildOrderedAlongWithRelationships creates item co-occurrence relationships
func (i *CSVImporter) buildOrderedAlongWithRelationships(ctx context.Context) error {
	query := `
		MATCH (o:Order)-[:HAS_ITEM]->(i1:Item)
		MATCH (o)-[:HAS_ITEM]->(i2:Item)
		WHERE i1.db_id < i2.db_id
		WITH i1, i2, count(o) as co_occurrences
		MERGE (i1)-[oaw1:ORDERED_ALONG_WITH]->(i2)
		SET oaw1.times = co_occurrences
		MERGE (i2)-[oaw2:ORDERED_ALONG_WITH]->(i1)
		SET oaw2.times = co_occurrences
		RETURN count(oaw1) as created_relationships
	`

	// Changed from ExecuteRead to ExecuteWrite
	err := i.client.ExecuteWrite(ctx, query, nil)
	if err != nil {
		return err
	}

	log.Println("Built ORDERED_ALONG_WITH relationships successfully")
	return nil
}

// clearDatabase removes all existing data (for development/testing)
func (i *CSVImporter) clearDatabase(ctx context.Context) error {
	query := `
		MATCH (n)
		DETACH DELETE n
		RETURN count(n) as deleted_nodes
	`

	log.Println("Clearing existing database...")
	return i.client.ExecuteWrite(ctx, query, nil)
}

// UpdateRelationshipsForNewOrder updates relationships when a new order is placed
func (i *CSVImporter) UpdateRelationshipsForNewOrder(ctx context.Context, orderID int) error {
	// Update HAS_ORDERED relationships for the new order
	updateHasOrderedQuery := `
		MATCH (o:Order {db_id: $orderID})-[hi:HAS_ITEM]->(i:Item)
		MATCH (o)<-[:HAS_MADE]-(u:User)
		WITH u, i, hi.quantity as quantity
		MERGE (u)-[ho:HAS_ORDERED]->(i)
		SET ho.times = COALESCE(ho.times, 0) + quantity
		RETURN count(ho) as updated_relationships
	`

	params := map[string]interface{}{
		"orderID": orderID,
	}

	if err := i.client.ExecuteWrite(ctx, updateHasOrderedQuery, params); err != nil {
		return fmt.Errorf("failed to update HAS_ORDERED relationships: %w", err)
	}

	// Update ORDERED_ALONG_WITH relationships for the new order
	updateCoOccurrenceQuery := `
		MATCH (o:Order {db_id: $orderID})-[:HAS_ITEM]->(i1:Item)
		MATCH (o)-[:HAS_ITEM]->(i2:Item)
		WHERE i1.db_id < i2.db_id
		WITH i1, i2
		MERGE (i1)-[oaw1:ORDERED_ALONG_WITH]->(i2)
		SET oaw1.times = COALESCE(oaw1.times, 0) + 1
		MERGE (i2)-[oaw2:ORDERED_ALONG_WITH]->(i1)
		SET oaw2.times = COALESCE(oaw2.times, 0) + 1
		RETURN count(oaw1) as updated_relationships
	`

	if err := i.client.ExecuteWrite(ctx, updateCoOccurrenceQuery, params); err != nil {
		return fmt.Errorf("failed to update ORDERED_ALONG_WITH relationships: %w", err)
	}

	return nil
}

// GetImportStatus returns the current state of the database
func (i *CSVImporter) GetImportStatus(ctx context.Context) (map[string]int, error) {
	query := `
		MATCH (u:User) WITH count(u) as users
		MATCH (i:Item) WITH users, count(i) as items
		MATCH (o:Order) WITH users, items, count(o) as orders
		MATCH ()-[ho:HAS_ORDERED]->() WITH users, items, orders, count(ho) as has_ordered
		MATCH ()-[oaw:ORDERED_ALONG_WITH]->() WITH users, items, orders, has_ordered, count(oaw) as ordered_along_with
		RETURN users, items, orders, has_ordered, ordered_along_with
	`

	results, err := i.client.ExecuteRead(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return map[string]int{
			"users":              0,
			"items":              0,
			"orders":             0,
			"has_ordered":        0,
			"ordered_along_with": 0,
		}, nil
	}

	result := results[0]
	status := map[string]int{
		"users":              int(result["users"].(int64)),
		"items":              int(result["items"].(int64)),
		"orders":             int(result["orders"].(int64)),
		"has_ordered":        int(result["has_ordered"].(int64)),
		"ordered_along_with": int(result["ordered_along_with"].(int64)),
	}

	return status, nil
}

// readCsvFile reads a CSV file and returns its records
func readCsvFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records from %s: %w", filePath, err)
	}

	return records, nil
}
