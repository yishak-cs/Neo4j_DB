import { useState, useEffect } from "react"
import { ShoppingCart, Star, TrendingUp, Clock, User as UserIcon, Loader2, Users } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import { RecommendationAPI, convertBackendRecommendation, handleApiCall, type Item, type User } from "./services/api"

// Types for the frontend
interface CartItem extends Item {
  quantity: number
}

interface Recommendation {
  item: Item
  score: number
  reason: string
  type: "frequent" | "co-order" | "global" | "trending"
}

export default function RestaurantApp() {
  const [cart, setCart] = useState<CartItem[]>([])
  const [selectedCategory, setSelectedCategory] = useState("All")
  const [recommendations, setRecommendations] = useState<Recommendation[]>([])
  const [items, setItems] = useState<Item[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [categories, setCategories] = useState<string[]>(["All"])
  const [isLoadingRecommendations, setIsLoadingRecommendations] = useState(false)
  const [isLoadingItems, setIsLoadingItems] = useState(false)
  const [isLoadingUsers, setIsLoadingUsers] = useState(false)
  const [isOnline, setIsOnline] = useState(false)
  const [selectedCartItem, setSelectedCartItem] = useState<Item | null>(null)

  // Check if backend is available and load initial data
  useEffect(() => {
    const initializeApp = async () => {
      const health = await handleApiCall(() => RecommendationAPI.healthCheck())
      const isHealthy = health !== null
      setIsOnline(isHealthy)

      if (isHealthy) {
        await loadUsers()
        await loadItems()
      }
    }
    initializeApp()
  }, [])

  // Load recommendations when user or cart changes
  useEffect(() => {
    if (selectedUser) {
      loadRecommendations()
    }
  }, [selectedUser, selectedCartItem])

  const loadUsers = async () => {
    setIsLoadingUsers(true)
    try {
      const response = await handleApiCall(() => RecommendationAPI.getAllUsers())
      if (response && response.users) {
        setUsers(response.users)
      }
    } catch (error) {
      console.error('Failed to load users:', error)
    } finally {
      setIsLoadingUsers(false)
    }
  }

  const loadItems = async () => {
    setIsLoadingItems(true)
    try {
      const response = await handleApiCall(() => RecommendationAPI.getAllItems())
      if (response && response.items) {
        setItems(response.items)
        
        // Extract unique categories
        const uniqueCategories = ["All", ...new Set(response.items.map(item => item.category))]
        setCategories(uniqueCategories)
      }
    } catch (error) {
      console.error('Failed to load items:', error)
    } finally {
      setIsLoadingItems(false)
    }
  }

  const loadRecommendations = async () => {
    if (!selectedUser || !isOnline) return

    setIsLoadingRecommendations(true)
    
    try {
      let recommendationsToShow: Recommendation[] = []

      if (selectedCartItem) {
        // First try to get user's co-orders for this item
        const userCoOrders = await handleApiCall(() => 
          RecommendationAPI.getUserCoOrderedItems(selectedUser.db_id, selectedCartItem.db_id)
        )
        
        if (userCoOrders && userCoOrders.recommendations && userCoOrders.recommendations.length > 0) {
          // User has co-orders with this item
          recommendationsToShow = userCoOrders.recommendations
            .map(convertBackendRecommendation)
            .filter(rec => !cart.some(cartItem => cartItem.db_id === rec.item.db_id))
            .slice(0, 4)
        } else {
          // No user co-orders, fall back to global co-orders
          const globalCoOrders = await handleApiCall(() => 
            RecommendationAPI.getGlobalCoOrderedItems(selectedCartItem.db_id)
          )
          
          if (globalCoOrders && globalCoOrders.recommendations) {
            recommendationsToShow = globalCoOrders.recommendations
              .map(convertBackendRecommendation)
              .filter(rec => !cart.some(cartItem => cartItem.db_id === rec.item.db_id))
              .slice(0, 4)
          }
        }
      } else {
        // No item selected, show user's frequent items
        const userFrequent = await handleApiCall(() => 
          RecommendationAPI.getUserFrequentItems(selectedUser.db_id)
        )
        
        if (userFrequent && userFrequent.recommendations) {
          recommendationsToShow = userFrequent.recommendations
            .map(convertBackendRecommendation)
            .filter(rec => !cart.some(cartItem => cartItem.db_id === rec.item.db_id))
            .slice(0, 4)
        }
      }
      
      setRecommendations(recommendationsToShow)
    } catch (error) {
      console.error('Failed to load recommendations:', error)
    } finally {
      setIsLoadingRecommendations(false)
    }
  }

  const addToCart = (item: Item) => {
    setCart((prev: CartItem[]) => {
      const existing = prev.find((cartItem: CartItem) => cartItem.db_id === item.db_id)
      if (existing) {
        return prev.map((cartItem: CartItem) =>
          cartItem.db_id === item.db_id ? { ...cartItem, quantity: cartItem.quantity + 1 } : cartItem,
        )
      }
      return [...prev, { ...item, quantity: 1 }]
    })

    // Set this item as selected for recommendations
    setSelectedCartItem(item)
  }

  const removeFromCart = (itemId: number) => {
    setCart((prev: CartItem[]) => {
      const newCart = prev.filter((item: CartItem) => item.db_id !== itemId)
      
      // Update selected cart item
      if (selectedCartItem?.db_id === itemId) {
        const remainingItem = newCart.length > 0 ? newCart[0] : null
        setSelectedCartItem(remainingItem)
      }
      
      return newCart
    })
  }

  const getTotalPrice = () => {
    return cart.reduce((total: number, item: CartItem) => total + item.price * item.quantity, 0)
  }

  const getRecommendationIcon = (type: string) => {
    switch (type) {
      case "frequent":
        return <UserIcon className="h-4 w-4" />
      case "co-order":
        return <Star className="h-4 w-4" />
      case "global":
        return <TrendingUp className="h-4 w-4" />
      case "trending":
        return <Clock className="h-4 w-4" />
      default:
        return <Star className="h-4 w-4" />
    }
  }

  const getRecommendationColor = (type: string) => {
    switch (type) {
      case "frequent":
        return "bg-blue-100 text-blue-800"
      case "co-order":
        return "bg-green-100 text-green-800"
      case "global":
        return "bg-purple-100 text-purple-800"
      case "trending":
        return "bg-orange-100 text-orange-800"
      default:
        return "bg-gray-100 text-gray-800"
    }
  }

  const filteredItems = selectedCategory === "All" ? items : items.filter(item => item.category === selectedCategory)

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-gray-900">NeoRestro</h1>
              {!isOnline && (
                <Badge variant="destructive" className="text-xs">
                  Offline Mode
                </Badge>
              )}
              {isOnline && (
                <Badge variant="outline" className="text-xs border-green-200 text-green-700">
                  Connected
                </Badge>
              )}
            </div>

            <div className="flex items-center gap-4">
              {/* User Selection */}
              <div className="flex items-center gap-2">
                <Users className="h-4 w-4 text-gray-500" />
                <select
                  value={selectedUser?.db_id || ""}
                  onChange={(e) => {
                    const userId = parseInt(e.target.value)
                    const user = users.find(u => u.db_id === userId)
                    setSelectedUser(user || null)
                    setRecommendations([]) // Clear recommendations when switching users
                    setSelectedCartItem(null) // Clear selected cart item
                  }}
                  className="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  disabled={isLoadingUsers || !isOnline}
                >
                  <option value="">
                    {isLoadingUsers ? "Loading users..." : "Select a user"}
                  </option>
                  {users.map((user) => (
                    <option key={user.db_id} value={user.db_id}>
                      {user.name}
                    </option>
                  ))}
                </select>
              </div>

              <Sheet>
                <SheetTrigger asChild>
                  <Button variant="outline" size="sm" className="relative bg-transparent">
                    <ShoppingCart className="h-4 w-4 mr-2" />
                    Cart ({cart.length})
                    {cart.length > 0 && (
                      <Badge className="absolute -top-2 -right-2 h-5 w-5 rounded-full p-0 flex items-center justify-center">
                        {cart.reduce((sum: number, item: CartItem) => sum + item.quantity, 0)}
                      </Badge>
                    )}
                  </Button>
                </SheetTrigger>
                <SheetContent>
                  <SheetHeader>
                    <SheetTitle>Your Order</SheetTitle>
                    <SheetDescription>Review your items before checkout</SheetDescription>
                  </SheetHeader>
                  <div className="mt-6 space-y-4">
                    {cart.length === 0 ? (
                      <p className="text-gray-500 text-center py-8">Your cart is empty</p>
                    ) : (
                      <>
                        {cart.map((item: CartItem) => (
                          <div key={item.db_id} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                            <div className="flex-1">
                              <h4 className="font-medium">{item.name}</h4>
                              <p className="text-sm text-gray-600">Qty: {item.quantity}</p>
                              {selectedCartItem?.db_id === item.db_id && (
                                <Badge variant="outline" className="text-xs mt-1">
                                  Getting recommendations for this item
                                </Badge>
                              )}
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium">${(item.price * item.quantity).toFixed(2)}</span>
                              <Button variant="ghost" size="sm" onClick={() => removeFromCart(item.db_id)}>
                                ×
                              </Button>
                            </div>
                          </div>
                        ))}
                        <Separator />
                        <div className="flex justify-between items-center font-bold text-lg">
                          <span>Total:</span>
                          <span>${getTotalPrice().toFixed(2)}</span>
                        </div>
                        <Button className="w-full" size="lg">
                          Checkout
                        </Button>
                      </>
                    )}
                  </div>
                </SheetContent>
              </Sheet>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* User Status */}
        {selectedUser && (
          <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center gap-2">
              <UserIcon className="h-5 w-5 text-blue-600" />
              <span className="font-medium text-blue-900">
                Showing recommendations for: {selectedUser.name}
              </span>
              {selectedCartItem && (
                <span className="text-blue-700">
                  • Based on: {selectedCartItem.name}
                </span>
              )}
            </div>
          </div>
        )}

        {/* Recommendations Section */}
        <section className="mb-12">
          <div className="flex items-center gap-2 mb-6">
            <Star className="h-5 w-5 text-yellow-500" />
            <h2 className="text-2xl font-bold text-gray-900">Recommended for You</h2>
            {isLoadingRecommendations && <Loader2 className="h-4 w-4 animate-spin text-gray-500" />}
          </div>
          
          {!selectedUser ? (
            <div className="text-center py-12 bg-white rounded-lg border-2 border-dashed border-gray-300">
              <Users className="h-12 w-12 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">Select a User to See Recommendations</h3>
              <p className="text-gray-500">
                Choose a user from the dropdown above to see personalized recommendations based on their order history.
              </p>
            </div>
          ) : recommendations.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              {recommendations.map((rec: Recommendation) => (
                <Card key={rec.item.db_id} className="hover:shadow-lg transition-shadow border-l-4 border-l-yellow-400">
                  <CardHeader className="pb-3">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-lg">{rec.item.name}</CardTitle>
                      <Badge className={getRecommendationColor(rec.type)}>
                        <div className="flex items-center gap-1">
                          {getRecommendationIcon(rec.type)}
                          <span className="text-xs">{Math.round(rec.score * 100)}%</span>
                        </div>
                      </Badge>
                    </div>
                    <CardDescription>{rec.reason}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex justify-between items-center">
                      <span className="text-2xl font-bold text-green-600">${rec.item.price}</span>
                      <Button onClick={() => addToCart(rec.item)} size="sm">
                        Add to Cart
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              {isLoadingRecommendations ? (
                <div className="flex items-center justify-center gap-2">
                  <Loader2 className="h-5 w-5 animate-spin" />
                  <span>Loading personalized recommendations...</span>
                </div>
              ) : isOnline ? (
                <div>
                  <p>No recommendations available for {selectedUser.name} right now.</p>
                  <p className="text-sm mt-1">Try adding items to your cart to get item-based suggestions!</p>
                </div>
              ) : (
                "Recommendations unavailable in offline mode. Connect to see personalized suggestions!"
              )}
            </div>
          )}
        </section>

        {/* Category Filter */}
        <div className="flex flex-wrap gap-2 mb-8">
          {categories.map((category: string) => (
            <Button
              key={category}
              variant={selectedCategory === category ? "default" : "outline"}
              onClick={() => setSelectedCategory(category)}
              size="sm"
              disabled={isLoadingItems}
            >
              {category}
            </Button>
          ))}
        </div>

        {/* Menu Items */}
        <section>
          <h2 className="text-2xl font-bold text-gray-900 mb-6">
            {selectedCategory === "All" ? "Full Menu" : selectedCategory}
            {isLoadingItems && <Loader2 className="inline h-5 w-5 ml-2 animate-spin" />}
          </h2>
          
          {isLoadingItems ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              {[...Array(8)].map((_, i) => (
                <Card key={i} className="animate-pulse">
                  <div className="aspect-video bg-gray-300 rounded-t-lg"></div>
                  <CardHeader>
                    <div className="h-4 bg-gray-300 rounded w-3/4"></div>
                    <div className="h-3 bg-gray-300 rounded w-1/2 mt-2"></div>
                  </CardHeader>
                  <CardContent>
                    <div className="flex justify-between items-center">
                      <div className="h-6 bg-gray-300 rounded w-1/3"></div>
                      <div className="h-8 bg-gray-300 rounded w-1/4"></div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : filteredItems.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              {filteredItems.map((item: Item) => (
                <Card key={item.db_id} className="hover:shadow-lg transition-shadow">
                  <div className="aspect-video bg-gradient-to-br from-orange-100 to-red-100 rounded-t-lg flex items-center justify-center">
                    <span className="text-4xl">🍽️</span>
                  </div>
                  <CardHeader>
                    <div className="flex justify-between items-start">
                      <CardTitle className="text-lg">{item.name}</CardTitle>
                      <Badge variant="secondary">{item.category}</Badge>
                    </div>
                    <CardDescription>{item.description}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex justify-between items-center">
                      <span className="text-2xl font-bold text-green-600">${item.price}</span>
                      <Button 
                        onClick={() => addToCart(item)}
                        disabled={!selectedUser}
                        title={!selectedUser ? "Select a user first" : ""}
                      >
                        Add to Cart
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              {isOnline ? "No items found in this category." : "Items unavailable in offline mode."}
            </div>
          )}
        </section>
      </div>
    </div>
  )
} 