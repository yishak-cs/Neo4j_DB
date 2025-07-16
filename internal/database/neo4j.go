package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jClient wraps the Neo4j driver with application-specific methods
type Neo4jClient struct {
	driver   neo4j.DriverWithContext
	database string
}

// Config holds the Neo4j connection configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string // typically "neo4j" for AuraDB
}

// NewNeo4jClient creates a new Neo4j client connection
func NewNeo4jClient(config Config) (*Neo4jClient, error) {
	// Configure driver with authentication
	driver, err := neo4j.NewDriverWithContext(config.URI, neo4j.BasicAuth(config.Username, config.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	log.Println("Successfully connected to Neo4j AuraDB")
	return &Neo4jClient{
		driver:   driver,
		database: config.Database,
	}, nil
}

// Close closes the Neo4j driver connection
func (c *Neo4jClient) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// ExecuteQuery executes a Cypher query and returns the result
func (c *Neo4jClient) ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	// Include database in session config
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// ExecuteWrite executes a write query (CREATE, MERGE, DELETE, etc.)
func (c *Neo4jClient) ExecuteWrite(ctx context.Context, query string, params map[string]interface{}) error {
	// Using the newer ExecuteQuery approach with write access mode
	_, err := neo4j.ExecuteQuery(
		ctx,
		c.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(c.database),
		neo4j.ExecuteQueryWithWritersRouting()) // This ensures write access

	if err != nil {
		return fmt.Errorf("failed to execute write query: %w", err)
	}

	return nil
}

// ExecuteWriteWithResult executes a write query and returns results
func (c *Neo4jClient) ExecuteWriteWithResult(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	result, err := neo4j.ExecuteQuery(
		ctx,
		c.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(c.database),
		neo4j.ExecuteQueryWithWritersRouting()) // This ensures write access

	if err != nil {
		return nil, fmt.Errorf("failed to execute write query: %w", err)
	}

	// Convert records to map slice
	var results []map[string]interface{}
	for _, record := range result.Records {
		recordMap := make(map[string]interface{})
		for i, key := range record.Keys {
			recordMap[key] = record.Values[i]
		}
		results = append(results, recordMap)
	}

	return results, nil
}

// ExecuteRead executes a read query and processes results
func (c *Neo4jClient) ExecuteRead(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	result, err := neo4j.ExecuteQuery(
		ctx,
		c.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(c.database),
		neo4j.ExecuteQueryWithReadersRouting()) // This ensures read access

	if err != nil {
		return nil, fmt.Errorf("failed to execute read query: %w", err)
	}

	// Convert records to map slice
	var results []map[string]interface{}
	for _, record := range result.Records {
		recordMap := make(map[string]interface{})
		for i, key := range record.Keys {
			recordMap[key] = record.Values[i]
		}
		results = append(results, recordMap)
	}

	return results, nil
}

// GetSession returns a new Neo4j session for complex operations
func (c *Neo4jClient) GetSession(ctx context.Context, accessMode neo4j.AccessMode) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   accessMode,
	})
}

// Health checks the database connection health
func (c *Neo4jClient) Health(ctx context.Context) error {
	_, err := neo4j.ExecuteQuery(
		ctx,
		c.driver,
		"RETURN 1",
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(c.database),
		neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// ExecuteWriteTransaction executes multiple write operations in a transaction
func (c *Neo4jClient) ExecuteWriteTransaction(ctx context.Context, work func(neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("failed to execute write transaction: %w", err)
	}

	return result, nil
}

// ExecuteReadTransaction executes multiple read operations in a transaction
func (c *Neo4jClient) ExecuteReadTransaction(ctx context.Context, work func(neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("failed to execute read transaction: %w", err)
	}

	return result, nil
}

// ExecuteWriteTransactionSimple executes a write transaction that doesn't need to return data
func (c *Neo4jClient) ExecuteWriteTransactionSimple(ctx context.Context, work func(neo4j.ManagedTransaction) error) error {
	wrappedWork := func(tx neo4j.ManagedTransaction) (interface{}, error) {
		err := work(tx)
		return nil, err
	}

	_, err := c.ExecuteWriteTransaction(ctx, wrappedWork)
	return err
}
