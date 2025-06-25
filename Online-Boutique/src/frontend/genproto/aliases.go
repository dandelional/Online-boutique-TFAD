package genproto

// 导入实际的包
import (
    hipstershop "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto/github.com/GoogleCloudPlatform/microservices-demo/hipstershop"
)

// 为所有需要的类型创建别名，这样 pb.xxx 就能正常工作了

// 基础类型
type Product = hipstershop.Product
type Money = hipstershop.Money
type Empty = hipstershop.Empty
type Cart = hipstershop.Cart
type CartItem = hipstershop.CartItem
type Address = hipstershop.Address

// 评分服务类型 - 这是关键部分
type SubmitRatingRequest = hipstershop.SubmitRatingRequest
type SubmitRatingResponse = hipstershop.SubmitRatingResponse
type GetProductRatingRequest = hipstershop.GetProductRatingRequest
type GetProductRatingResponse = hipstershop.GetProductRatingResponse
type GetUserRatingRequest = hipstershop.GetUserRatingRequest
type GetUserRatingResponse = hipstershop.GetUserRatingResponse
type Rating = hipstershop.Rating
type RatingServiceClient = hipstershop.RatingServiceClient

// 服务客户端构造函数
var NewRatingServiceClient = hipstershop.NewRatingServiceClient
var NewProductCatalogServiceClient = hipstershop.NewProductCatalogServiceClient
var NewCartServiceClient = hipstershop.NewCartServiceClient
var NewRecommendationServiceClient = hipstershop.NewRecommendationServiceClient
var NewShippingServiceClient = hipstershop.NewShippingServiceClient
var NewCheckoutServiceClient = hipstershop.NewCheckoutServiceClient
var NewCurrencyServiceClient = hipstershop.NewCurrencyServiceClient
var NewAdServiceClient = hipstershop.NewAdServiceClient

// 其他常用类型
type ListProductsResponse = hipstershop.ListProductsResponse
type GetProductRequest = hipstershop.GetProductRequest
type SearchProductsRequest = hipstershop.SearchProductsRequest
type SearchProductsResponse = hipstershop.SearchProductsResponse
type GetCartRequest = hipstershop.GetCartRequest
type AddItemRequest = hipstershop.AddItemRequest
type EmptyCartRequest = hipstershop.EmptyCartRequest
type ListRecommendationsRequest = hipstershop.ListRecommendationsRequest
type ListRecommendationsResponse = hipstershop.ListRecommendationsResponse
type GetSupportedCurrenciesResponse = hipstershop.GetSupportedCurrenciesResponse
type CurrencyConversionRequest = hipstershop.CurrencyConversionRequest
type GetQuoteRequest = hipstershop.GetQuoteRequest
type GetQuoteResponse = hipstershop.GetQuoteResponse
type ShipOrderRequest = hipstershop.ShipOrderRequest
type ShipOrderResponse = hipstershop.ShipOrderResponse
type PlaceOrderRequest = hipstershop.PlaceOrderRequest
type PlaceOrderResponse = hipstershop.PlaceOrderResponse
type OrderResult = hipstershop.OrderResult
type OrderItem = hipstershop.OrderItem
type CreditCardInfo = hipstershop.CreditCardInfo
type ChargeRequest = hipstershop.ChargeRequest
type ChargeResponse = hipstershop.ChargeResponse
type AdRequest = hipstershop.AdRequest
type AdResponse = hipstershop.AdResponse
type Ad = hipstershop.Ad

// 服务客户端接口
type ProductCatalogServiceClient = hipstershop.ProductCatalogServiceClient
type CartServiceClient = hipstershop.CartServiceClient
type RecommendationServiceClient = hipstershop.RecommendationServiceClient
type ShippingServiceClient = hipstershop.ShippingServiceClient
type CheckoutServiceClient = hipstershop.CheckoutServiceClient
type CurrencyServiceClient = hipstershop.CurrencyServiceClient
type PaymentServiceClient = hipstershop.PaymentServiceClient
type EmailServiceClient = hipstershop.EmailServiceClient
type AdServiceClient = hipstershop.AdServiceClient