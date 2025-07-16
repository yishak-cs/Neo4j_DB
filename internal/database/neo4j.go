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
	driver neo4j.DriverWithContext
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
	return &Neo4jClient{driver: driver}, nil
}

// Close closes the Neo4j driver connection
func (c *Neo4jClient) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// ExecuteQuery executes a Cypher query and returns the result
func (c *Neo4jClient) ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// ExecuteWrite executes a write query (CREATE, MERGE, DELETE, etc.)
func (c *Neo4jClient) ExecuteWrite(ctx context.Context, query string, params map[string]interface{}) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to execute write query: %w", err)
	}

	return nil
}

// ExecuteRead executes a read query and processes results
func (c *Neo4jClient) ExecuteRead(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	var results []map[string]interface{}

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			record := result.Record()
			recordMap := make(map[string]interface{})

			for _, key := range record.Keys {
				recordMap[key] = record.Values[len(recordMap)]
			}
			results = append(results, recordMap)
		}

		return result.Consume(ctx)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute read query: %w", err)
	}

	return results, nil
}

// GetSession returns a new Neo4j session for complex operations
func (c *Neo4jClient) GetSession(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{})
}

// Health checks the database connection health
func (c *Neo4jClient) Health(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "RETURN 1", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	_, err = result.Consume(ctx)
	if err != nil {
		return fmt.Errorf("health check failed to consume result: %w", err)
	}

	return nil
}
