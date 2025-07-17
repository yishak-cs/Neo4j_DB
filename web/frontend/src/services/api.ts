import axios from 'axios'

// Types matching the Go backend models
export interface User {
  db_id: number
  name: string
  email: string
  created_at?: string
}

export interface Item {
  db_id: number
  name: string
  price: number
  category: string
  description?: string
}

export interface Recommendation {
  item: Item
  score: number
  explanation: string
  strategy: string
}

export interface HybridWeights {
  user_frequency: number
  user_co_orders: number
  global_co_orders: number
  time_based_trend: number
}

export interface RecommendationResponse {
  user_id?: number
  item_id?: number
  item_in_cart?: number
  weights?: HybridWeights
  recommendations: Recommendation[]
  strategy: string
  description: string
}

// Create axios instance with default config
const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

export class RecommendationAPI {
  // Get all users
  static async getAllUsers(): Promise<{ users: User[], count: number }> {
    const response = await api.get('/users')
    return response.data
  }

  // Get all menu items
  static async getAllItems(): Promise<{ items: Item[], count: number }> {
    const response = await api.get('/items')
    return response.data
  }

  // Get user's most frequently ordered items
  static async getUserFrequentItems(userId: number): Promise<RecommendationResponse> {
    const response = await api.get(`/recommendations/user-frequent/${userId}`)
    return response.data
  }

  // Get items user frequently orders with a specific item
  static async getUserCoOrderedItems(userId: number, itemId: number): Promise<RecommendationResponse> {
    const response = await api.get(`/recommendations/user-co-orders/${userId}/${itemId}`)
    return response.data
  }

  // Get items frequently ordered with a specific item by all users
  static async getGlobalCoOrderedItems(itemId: number): Promise<RecommendationResponse> {
    const response = await api.get(`/recommendations/global-co-orders/${itemId}`)
    return response.data
  }

  // Get trending items
  static async getTrendingItems(days: number = 7): Promise<RecommendationResponse> {
    const response = await api.get(`/recommendations/trending?days=${days}`)
    return response.data
  }

  // Get hybrid recommendations
  static async getHybridRecommendations(
    userId: number,
    itemInCart?: number,
    weights?: Partial<HybridWeights>
  ): Promise<RecommendationResponse> {
    const params = new URLSearchParams()
    
    if (itemInCart) {
      params.append('itemInCart', itemInCart.toString())
    }
    
    if (weights) {
      if (weights.user_frequency !== undefined) {
        params.append('userFreq', weights.user_frequency.toString())
      }
      if (weights.user_co_orders !== undefined) {
        params.append('userCoOrders', weights.user_co_orders.toString())
      }
      if (weights.global_co_orders !== undefined) {
        params.append('globalCoOrders', weights.global_co_orders.toString())
      }
      if (weights.time_based_trend !== undefined) {
        params.append('timeTrend', weights.time_based_trend.toString())
      }
    }

    const queryString = params.toString()
    const url = `/recommendations/hybrid/${userId}${queryString ? `?${queryString}` : ''}`
    
    const response = await api.get(url)
    return response.data
  }

  // Check server health
  static async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await api.get('/health')
    return response.data
  }
}

// Utility function to convert backend recommendation to frontend format
export function convertBackendRecommendation(backendRec: Recommendation): {
  item: Item
  score: number
  reason: string
  type: "frequent" | "co-order" | "global" | "trending"
} {
  let type: "frequent" | "co-order" | "global" | "trending" = "global"
  
  switch (backendRec.strategy) {
    case "UserFrequency":
      type = "frequent"
      break
    case "UserCoOrders":
      type = "co-order"
      break
    case "GlobalCoOrders":
      type = "global"
      break
    case "TimeBasedTrend":
      type = "trending"
      break
  }

  return {
    item: backendRec.item,
    score: backendRec.score,
    reason: backendRec.explanation,
    type: type
  }
}

// Error handling wrapper
export async function handleApiCall<T>(apiCall: () => Promise<T>): Promise<T | null> {
  try {
    return await apiCall()
  } catch (error) {
    console.error('API call failed:', error)
    
    if (axios.isAxiosError(error)) {
      if (error.response) {
        console.error('Response error:', error.response.data)
      } else if (error.request) {
        console.error('Request error - no response received')
      }
    }
    
    return null
  }
} 