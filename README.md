# NeoRestro Recommendation Engine

A restaurant recommendation engine built with Neo4j and Go.

## Project Overview

NeoRestro is a recommendation engine that uses graph database technology (Neo4j) to provide personalized menu item recommendations for restaurant customers. It implements four recommendation strategies:

1. **User Frequency**: What does a user generally order most frequently?
2. **User Co-Orders**: Once item X is in cart, what did THIS user previously order with X?
3. **Global Co-Orders**: Once item X is in cart, what items are frequently ordered with X across ALL users?
4. **Time-Based Trend**: What items are trending recently?

## Architecture

- **Database**: Neo4j AuraDB (cloud-hosted)
- **Backend**: Go web application with Gin framework
- **Data Import**: CSV files imported via Cypher LOAD CSV

## Data Model

### Nodes
- User {db_id, name, email, created_at}
- Item {db_id, name, price, category}
- Order {db_id, created_at, total_amount}

### Relationships
- User -[:HAS_ORDERED {times}]-> Item (aggregated frequency)
- User -[:HAS_MADE]-> Order (user's orders)
- Order -[:HAS_ITEM {quantity}]-> Item (order contents)
- Item -[:ORDERED_ALONG_WITH {times}]-> Item (co-occurrence)

## Setup

### Prerequisites
- Go 1.16+
- Neo4j AuraDB instance

### Environment Variables
Create a `.env` file with:

```
NEO4J_URI=neo4j+s://<your-instance-id>.databases.neo4j.io
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your-password-here
APP_PORT=8080
```

### Running the Application

1. Install dependencies:
   ```
   go mod download
   ```

2. Run the server:
   ```
   go run cmd/server/main.go
   ```

3. Import sample data:
   ```
   curl -X POST http://localhost:8080/import
   ```

## API Endpoints

### Health Check
- `GET /health` - Check service health

### Data Import
- `POST /import` - Import data from CSV files

### Recommendations
- `GET /api/recommendations/user-frequent/:userId` - Get user's most frequently ordered items
- `GET /api/recommendations/user-co-orders/:userId/:itemId` - Get items a user frequently orders with a specific item
- `GET /api/recommendations/global-co-orders/:itemId` - Get items frequently ordered with a specific item by all users
- `GET /api/recommendations/trending` - Get currently trending items
- `GET /api/recommendations/hybrid/:userId` - Get personalized hybrid recommendations

#### Hybrid Recommendations Parameters
- `itemInCart` - Optional item ID in the cart
- `userFreq` - Weight for user frequency (default varies by user experience)
- `userCoOrders` - Weight for user co-orders
- `globalCoOrders` - Weight for global co-orders
- `timeTrend` - Weight for time-based trends

## Example Usage

```bash
# Get hybrid recommendations for user 1 with item 6 in cart
curl http://localhost:8080/api/recommendations/hybrid/1?itemInCart=6

# Get user's most frequently ordered items
curl http://localhost:8080/api/recommendations/user-frequent/2

# Get items frequently ordered with item 1 by all users
curl http://localhost:8080/api/recommendations/global-co-orders/1

# Get trending items from the last 7 days
curl http://localhost:8080/api/recommendations/trending
``` 