# NeoRestro Setup Guide

## Neo4j AuraDB Configuration

1. **Create a .env file** in the project root with the following variables:

```env
# Neo4j AuraDB Configuration
NEO4J_URI=your_auradb_uri_here
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your_password_here
NEO4J_DATABASE=neo4j
APP_PORT=8080
```

2. **Replace with your actual AuraDB details:**
   - `NEO4J_URI`: Your AuraDB connection URI (e.g., `neo4j+s://12345678.databases.neo4j.io`)
   - `NEO4J_USERNAME`: Usually "neo4j"
   - `NEO4J_PASSWORD`: The password you set when creating your AuraDB instance
   - `NEO4J_DATABASE`: Usually "neo4j" for AuraDB

## Running the Application

### Backend (Go Server)

1. **Install Go dependencies:**
```bash
go mod download
```

2. **Import CSV data to Neo4j:**
```bash
go run cmd/server/main.go
# Then visit: http://localhost:8080/import
```

3. **Start the server:**
```bash
go run cmd/server/main.go
```

The server will run on `http://localhost:8080`

### Frontend (React App)

1. **Navigate to frontend directory:**
```bash
cd web/frontend
```

2. **Install dependencies:**
```bash
pnpm install
```

3. **Start development server:**
```bash
pnpm dev
```

The frontend will run on `http://localhost:3000`

4. **Build for production:**
```bash
pnpm build
```

This builds the frontend into `web/static/` which is served by the Go server.

## API Endpoints

### Menu Items
- `GET /api/items` - Get all menu items
- `GET /api/items/category/:category` - Get items by category

### Recommendations
- `GET /api/recommendations/user-frequent/:userId` - User's frequent items
- `GET /api/recommendations/user-co-orders/:userId/:itemId` - User's co-ordered items
- `GET /api/recommendations/global-co-orders/:itemId` - Global co-ordered items
- `GET /api/recommendations/trending?days=7` - Trending items
- `GET /api/recommendations/hybrid/:userId?itemInCart=123` - Hybrid recommendations

### Utility
- `GET /api/health` - Health check
- `POST /api/import` - Import CSV data

## Data Model

The system uses these CSV files:
- `data/users.csv` - Customer information
- `data/items.csv` - Menu items
- `data/orders.csv` - Order history
- `data/order_items.csv` - Order-item relationships

## Recommendation Strategies

1. **UserFrequency** - Items the user orders most frequently
2. **UserCoOrders** - Items the user previously ordered with current cart item
3. **GlobalCoOrders** - Items all users frequently order together
4. **TimeBasedTrend** - Currently trending items

The hybrid system combines all strategies with intelligent weighting based on user experience. 