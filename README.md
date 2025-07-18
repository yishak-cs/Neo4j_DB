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
- [pnpm](https://pnpm.io/) (for frontend)

### Environment Variables
Create a `.env` file with:

```
NEO4J_URI=neo4j+s://<your-instance-id>.databases.neo4j.io
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your-password-here
APP_PORT=8080
```

### Running the Application

#### Frontend

1. Open a terminal and navigate to the frontend directory:
   ```
   cd web/frontend
   ```
2. Install dependencies:
   ```
   pnpm install
   ```
3. Build the frontend:
   ```
   pnpm build
   ```
4. Start the development server:
   ```
   pnpm dev
   ```
5. Open your browser and go to [http://localhost:3000](http://localhost:3000)

#### Backend (optional, if you want to run the Go server)

1. Install dependencies:
   ```
   go mod download
   ```
2. Run the server:
   ```
   go run cmd/server/main.go
   ```

#### Data Import (optional, if you want to import sample data)

1. Import sample data (see your backend API or scripts for details).

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

To use the frontend, follow the instructions above and visit [http://localhost:3000](http://localhost:3000). 