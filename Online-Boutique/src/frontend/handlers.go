// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"github.com/GoogleCloudPlatform/microservices-demo/src/frontend/money"
	"github.com/GoogleCloudPlatform/microservices-demo/src/frontend/validator"
)

type platformDetails struct {
	css      string
	provider string
}

var (
	frontendMessage  = strings.TrimSpace(os.Getenv("FRONTEND_MESSAGE"))
	isCymbalBrand    = "true" == strings.ToLower(os.Getenv("CYMBAL_BRANDING"))
	assistantEnabled = "true" == strings.ToLower(os.Getenv("ENABLE_ASSISTANT"))
	templates        = template.Must(template.New("").
            Funcs(template.FuncMap{
        "renderMoney":        renderMoney,
        "renderCurrencyLogo": renderCurrencyLogo,
        "until": func(n int) []int {
            result := make([]int, n)
            for i := 0; i < n; i++ {
                result[i] = i
            }
            return result
        },
        "int": func(value interface{}) int {
            switch v := value.(type) {
            case int:
                return v
            case int32:
                return int(v)
            case int64:
                return int(v)
            case float32:
                return int(v)
            case float64:
                return int(v)
            case string:
                if i, err := strconv.Atoi(v); err == nil {
                    return i
                }
                return 0
            default:
                return 0
            }
        },
        "add": func(a, b int) int {
            return a + b
        },
        "sub": func(a, b int) int {
            return a - b
        },
        "mul": func(a, b int) int {
            return a * b
        },
        "div": func(a, b int) int {
            if b != 0 {
                return a / b
            }
            return 0
        },
        "eq": func(a, b interface{}) bool {
            return a == b
        },
        "ne": func(a, b interface{}) bool {
            return a != b
        },
        "lt": func(a, b interface{}) bool {
            switch va := a.(type) {
            case int:
                if vb, ok := b.(int); ok {
                    return va < vb
                }
            case int32:
                if vb, ok := b.(int32); ok {
                    return va < vb
                }
            case float32:
                if vb, ok := b.(float32); ok {
                    return va < vb
                }
            case float64:
                if vb, ok := b.(float64); ok {
                    return va < vb
                }
            }
            return false
        },
        "gt": func(a, b interface{}) bool {
            switch va := a.(type) {
            case int:
                if vb, ok := b.(int); ok {
                    return va > vb
                }
            case int32:
                if vb, ok := b.(int32); ok {
                    return va > vb
                }
            case float32:
                if vb, ok := b.(float32); ok {
                    return va > vb
                }
            case float64:
                if vb, ok := b.(float64); ok {
                    return va > vb
                }
            }
            return false
        },
        "dict": func(values ...interface{}) (map[string]interface{}, error) {
            if len(values)%2 != 0 {
                return nil, fmt.Errorf("invalid dict call")
            }
            dict := make(map[string]interface{}, len(values)/2)
            for i := 0; i < len(values); i += 2 {
                key, ok := values[i].(string)
                if !ok {
                    return nil, fmt.Errorf("dict keys must be strings")
                }
                dict[key] = values[i+1]
            }
            return dict, nil
        },
    }).ParseGlob("templates/*.html"))
	plat platformDetails
)

var validEnvs = []string{"local", "gcp", "azure", "aws", "onprem", "alibaba"}

func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.WithField("currency", currentCurrency(r)).Info("home")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	products, err := fe.getProducts(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve products"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *pb.Product
		Price *pb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "failed to do currency conversion for product %s", p.GetId()), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	// Set ENV_PLATFORM (default to local if not set; use env var if set; otherwise detect GCP, which overrides env)_
	var env = os.Getenv("ENV_PLATFORM")
	// Only override from env variable if set + valid env
	if env == "" || stringinSlice(validEnvs, env) == false {
		fmt.Println("env platform is either empty or invalid")
		env = "local"
	}
	// Autodetect GCP
	addrs, err := net.LookupHost("metadata.google.internal.")
	if err == nil && len(addrs) >= 0 {
		log.Debugf("Detected Google metadata server: %v, setting ENV_PLATFORM to GCP.", addrs)
		env = "gcp"
	}

	log.Debugf("ENV_PLATFORM is: %s", env)
	plat = platformDetails{}
	plat.setPlatformDetails(strings.ToLower(env))

	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"cart_size":     cartSize(cart),
		"banner_color":  os.Getenv("BANNER_COLOR"), // illustrates canary deployments
		"ad":            fe.chooseAd(r.Context(), []string{}, log),
	})); err != nil {
		log.Error(err)
	}
}

func (plat *platformDetails) setPlatformDetails(env string) {
	if env == "aws" {
		plat.provider = "AWS"
		plat.css = "aws-platform"
	} else if env == "onprem" {
		plat.provider = "On-Premises"
		plat.css = "onprem-platform"
	} else if env == "azure" {
		plat.provider = "Azure"
		plat.css = "azure-platform"
	} else if env == "gcp" {
		plat.provider = "Google Cloud"
		plat.css = "gcp-platform"
	} else if env == "alibaba" {
		plat.provider = "Alibaba Cloud"
		plat.css = "alibaba-platform"
	} else {
		plat.provider = "local"
		plat.css = "local"
	}
}

// func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
//     log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
//     id := mux.Vars(r)["id"]
//     if id == "" {
//         renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
//         return
//     }
//     log.WithField("id", id).WithField("currency", currentCurrency(r)).
//         Debug("serving product page")

//     p, err := fe.getProduct(r.Context(), id)
//     if err != nil {
//         renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
//         return
//     }
//     currencies, err := fe.getCurrencies(r.Context())
//     if err != nil {
//         renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
//         return
//     }

//     cart, err := fe.getCart(r.Context(), sessionID(r))
//     if err != nil {
//         renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
//         return
//     }

//     price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
//     if err != nil {
//         renderHTTPError(log, r, w, errors.Wrap(err, "failed to convert currency"), http.StatusInternalServerError)
//         return
//     }

//     recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), []string{id})
//     if err != nil {
//         log.WithField("error", err).Warn("failed to get product recommendations")
//     }

//     product := struct {
//         Item  *pb.Product
//         Price *pb.Money
//     }{p, price}

//     var packagingInfo *PackagingInfo = nil
//     if isPackagingServiceConfigured() {
//         packagingInfo, err = httpGetPackagingInfo(id)
//         if err != nil {
//             fmt.Println("Failed to obtain product's packaging info:", err)
//         }
//     }

//     // 获取产品评分
//     var productRating *pb.GetProductRatingResponse
//     if fe.ratingServiceClient != nil {
//         rating, err := fe.ratingServiceClient.GetProductRating(r.Context(), &pb.GetProductRatingRequest{
//             ProductId: id,
//         })
//         if err != nil {
//             log.Printf("Failed to get product rating: %v", err)
//         } else {
//             productRating = rating
//         }
//     }

//     // 检查用户是否已评分
//     var userRating *pb.GetUserRatingResponse
//     userID := sessionID(r)
//     if fe.ratingServiceClient != nil && userID != "" {
//         userRat, err := fe.ratingServiceClient.GetUserRating(r.Context(), &pb.GetUserRatingRequest{
//             ProductId: id,
//             UserId:    userID,
//         })
//         if err != nil {
//             log.Printf("Failed to get user rating: %v", err)
//         } else {
//             userRating = userRat
//         }
//     }

//     if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
//         "ad":              fe.chooseAd(r.Context(), p.Categories, log),
//         "show_currency":   true,
//         "currencies":      currencies,
//         "product":         product,
//         "recommendations": recommendations,
//         "cart_size":       cartSize(cart),
//         "packagingInfo":   packagingInfo,
//         "product_rating":  productRating,
//         "user_rating":     userRating,
//     })); err != nil {
//         log.Println(err)
//     }
// }

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
    log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
    id := mux.Vars(r)["id"]
    if id == "" {
        renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
        return
    }

    // 创建模拟产品数据，用于测试评分功能
    var product *pb.Product
    
    // 尝试从产品目录服务获取产品，如果失败则使用模拟数据
    if fe.productCatalogSvcConn != nil {
        catalog := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn)
        if p, err := catalog.GetProduct(r.Context(), &pb.GetProductRequest{Id: id}); err == nil {
            product = p
        } else {
            log.WithField("error", err).Warn("could not retrieve product, using mock data")
        }
    }
    
    // 如果没有获取到产品，使用模拟数据
    if product == nil {
        product = createMockProduct(id)
    }

    // 模拟价格转换
    price := &pb.Money{
        CurrencyCode: currentCurrency(r),
        Units:        product.PriceUsd.Units,
        Nanos:        product.PriceUsd.Nanos,
    }

    // 创建产品视图结构，匹配模板期望的格式
    productView := struct {
        Item  *pb.Product
        Price *pb.Money
    }{product, price}

    // 获取评分数据
    var productRating *pb.GetProductRatingResponse
    if fe.ratingServiceClient != nil {
        if rating, err := fe.ratingServiceClient.GetProductRating(r.Context(), &pb.GetProductRatingRequest{
            ProductId: id,
        }); err == nil {
            productRating = rating
        } else {
            log.WithField("error", err).Warn("could not retrieve product rating")
            // 创建默认评分响应
            productRating = &pb.GetProductRatingResponse{
                ProductId:    id,
                AverageScore: 0,
                TotalRatings: 0,
                Ratings:      []*pb.Rating{},
            }
        }
    }

    // 模拟其他数据
    currencies := []string{"USD", "EUR", "CAD", "JPY", "GBP", "TRY"}
    cart := []*pb.Product{}
    recommendations := []*pb.Product{}

    if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
        "show_currency":    true,
        "currencies":       currencies,
        "product":          productView,  // 使用正确的结构体
        "recommendations":  recommendations,
        "cart_size":        len(cart),
        "product_rating":   productRating,
        "ad":              nil, // 暂时设为 nil
    })); err != nil {
        log.Println(err)
    }
}

// 创建模拟产品数据的辅助函数
func createMockProduct(id string) *pb.Product {
    products := map[string]*pb.Product{
        "OLJCESPC7Z": {
            Id:          "OLJCESPC7Z",
            Name:        "数码相机",
            Description: "高质量数码相机，适合摄影爱好者",
            Picture:     "/static/img/products/camera.jpg",
            PriceUsd: &pb.Money{
                CurrencyCode: "USD",
                Units:        199,
                Nanos:        990000000,
            },
            Categories: []string{"photography", "electronics"},
        },
        "66VCHSJNUP": {
            Id:          "66VCHSJNUP",
            Name:        "经典T恤",
            Description: "舒适的棉质T恤，多种颜色可选",
            Picture:     "/static/img/products/t-shirt.jpg",
            PriceUsd: &pb.Money{
                CurrencyCode: "USD",
                Units:        29,
                Nanos:        990000000,
            },
            Categories: []string{"clothing"},
        },
        "1YMWWN1N4O": {
            Id:          "1YMWWN1N4O",
            Name:        "咖啡马克杯",
            Description: "精美陶瓷马克杯，容量350ml",
            Picture:     "/static/img/products/mug.jpg",
            PriceUsd: &pb.Money{
                CurrencyCode: "USD",
                Units:        15,
                Nanos:        990000000,
            },
            Categories: []string{"kitchen", "drinkware"},
        },
    }
    
    if product, exists := products[id]; exists {
        return product
    }
    
    // 默认产品
    return &pb.Product{
        Id:          id,
        Name:        "测试产品",
        Description: "这是一个用于测试评分功能的模拟产品",
        Picture:     "/static/img/products/default.jpg",
        PriceUsd: &pb.Money{
            CurrencyCode: "USD",
            Units:        99,
            Nanos:        990000000,
        },
        Categories: []string{"test"},
    }
}

func (fe *frontendServer) submitRatingHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    productID := r.FormValue("product_id")
    scoreStr := r.FormValue("score")
    comment := r.FormValue("comment")
    userID := sessionID(r)

    if productID == "" || scoreStr == "" || userID == "" {
        http.Error(w, "Missing required fields", http.StatusBadRequest)
        return
    }

    score, err := strconv.Atoi(scoreStr)
    if err != nil || score < 1 || score > 5 {
        http.Error(w, "Invalid score", http.StatusBadRequest)
        return
    }

    if fe.ratingServiceClient != nil {
        _, err := fe.ratingServiceClient.SubmitRating(r.Context(), &pb.SubmitRatingRequest{
            ProductId: productID,
            UserId:    userID,
            Score:     int32(score),
            Comment:   comment,
        })
        if err != nil {
            log.Printf("Failed to submit rating: %v", err)
            http.Error(w, "Failed to submit rating", http.StatusInternalServerError)
            return
        }
    }

    http.Redirect(w, r, baseUrl+"/product/"+productID, http.StatusSeeOther)
}

func (fe *frontendServer) addToCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	quantity, _ := strconv.ParseUint(r.FormValue("quantity"), 10, 32)
	productID := r.FormValue("product_id")
	payload := validator.AddToCartPayload{
		Quantity:  quantity,
		ProductID: productID,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("product", payload.ProductID).WithField("quantity", payload.Quantity).Debug("adding to cart")

	p, err := fe.getProduct(r.Context(), payload.ProductID)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
		return
	}

	if err := fe.insertCart(r.Context(), sessionID(r), p.GetId(), int32(payload.Quantity)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to add to cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl + "/cart")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) emptyCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("emptying cart")

	if err := fe.emptyCart(r.Context(), sessionID(r)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to empty cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl + "/")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("view user cart")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	// ignores the error retrieving recommendations since it is not critical
	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), cartIDs(cart))
	if err != nil {
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	shippingCost, err := fe.getShippingQuote(r.Context(), cart, currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to get shipping quote"), http.StatusInternalServerError)
		return
	}

	type cartItemView struct {
		Item     *pb.Product
		Quantity int32
		Price    *pb.Money
	}
	items := make([]cartItemView, len(cart))
	totalPrice := pb.Money{CurrencyCode: currentCurrency(r)}
	for i, item := range cart {
		p, err := fe.getProduct(r.Context(), item.GetProductId())
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not retrieve product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not convert currency for product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}

		multPrice := money.MultiplySlow(*price, uint32(item.GetQuantity()))
		items[i] = cartItemView{
			Item:     p,
			Quantity: item.GetQuantity(),
			Price:    &multPrice}
		totalPrice = money.Must(money.Sum(totalPrice, multPrice))
	}
	totalPrice = money.Must(money.Sum(totalPrice, *shippingCost))
	year := time.Now().Year()

	if err := templates.ExecuteTemplate(w, "cart", injectCommonTemplateData(r, map[string]interface{}{
		"currencies":       currencies,
		"recommendations":  recommendations,
		"cart_size":        cartSize(cart),
		"shipping_cost":    shippingCost,
		"show_currency":    true,
		"total_cost":       totalPrice,
		"items":            items,
		"expiration_years": []int{year, year + 1, year + 2, year + 3, year + 4},
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("placing order")

	var (
		email         = r.FormValue("email")
		streetAddress = r.FormValue("street_address")
		zipCode, _    = strconv.ParseInt(r.FormValue("zip_code"), 10, 32)
		city          = r.FormValue("city")
		state         = r.FormValue("state")
		country       = r.FormValue("country")
		ccNumber      = r.FormValue("credit_card_number")
		ccMonth, _    = strconv.ParseInt(r.FormValue("credit_card_expiration_month"), 10, 32)
		ccYear, _     = strconv.ParseInt(r.FormValue("credit_card_expiration_year"), 10, 32)
		ccCVV, _      = strconv.ParseInt(r.FormValue("credit_card_cvv"), 10, 32)
	)

	payload := validator.PlaceOrderPayload{
		Email:         email,
		StreetAddress: streetAddress,
		ZipCode:       zipCode,
		City:          city,
		State:         state,
		Country:       country,
		CcNumber:      ccNumber,
		CcMonth:       ccMonth,
		CcYear:        ccYear,
		CcCVV:         ccCVV,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}

	order, err := pb.NewCheckoutServiceClient(fe.checkoutSvcConn).
		PlaceOrder(r.Context(), &pb.PlaceOrderRequest{
			Email: payload.Email,
			CreditCard: &pb.CreditCardInfo{
				CreditCardNumber:          payload.CcNumber,
				CreditCardExpirationMonth: int32(payload.CcMonth),
				CreditCardExpirationYear:  int32(payload.CcYear),
				CreditCardCvv:             int32(payload.CcCVV)},
			UserId:       sessionID(r),
			UserCurrency: currentCurrency(r),
			Address: &pb.Address{
				StreetAddress: payload.StreetAddress,
				City:          payload.City,
				State:         payload.State,
				ZipCode:       int32(payload.ZipCode),
				Country:       payload.Country},
		})
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to complete the order"), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("order placed")

	order.GetOrder().GetItems()
	recommendations, _ := fe.getRecommendations(r.Context(), sessionID(r), nil)

	totalPaid := *order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(*v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "order", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency":   false,
		"currencies":      currencies,
		"order":           order.GetOrder(),
		"total_paid":      &totalPaid,
		"recommendations": recommendations,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) assistantHandler(w http.ResponseWriter, r *http.Request) {
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "assistant", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"currencies":    currencies,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("logging out")
	for _, c := range r.Cookies() {
		c.Expires = time.Now().Add(-time.Hour * 24 * 365)
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", baseUrl + "/")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) getProductByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["ids"]
	if id == "" {
		return
	}

	p, err := fe.getProduct(r.Context(), id)
	if err != nil {
		return
	}

	jsonData, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err)
		return
	}

	w.Write(jsonData)
	w.WriteHeader(http.StatusOK)
}

func (fe *frontendServer) chatBotHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	type Response struct {
		Message string `json:"message"`
	}

	type LLMResponse struct {
		Content string         `json:"content"`
		Details map[string]any `json:"details"`
	}

	var response LLMResponse

	url := "http://" + fe.shoppingAssistantSvcAddr
	req, err := http.NewRequest(http.MethodPost, url, r.Body)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to create request"), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to send request"), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to read response"), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%+v\n", body)
	fmt.Printf("%+v\n", res)

	err = json.Unmarshal(body, &response)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to unmarshal body"), http.StatusInternalServerError)
		return
	}

	// respond with the same message
	json.NewEncoder(w).Encode(Response{Message: response.Content})

	w.WriteHeader(http.StatusOK)
}

func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	cur := r.FormValue("currency_code")
	payload := validator.SetCurrencyPayload{Currency: cur}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("curr.new", payload.Currency).WithField("curr.old", currentCurrency(r)).
		Debug("setting currency")

	if payload.Currency != "" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieCurrency,
			Value:  payload.Currency,
			MaxAge: cookieMaxAge,
		})
	}
	referer := r.Header.Get("referer")
	if referer == "" {
		referer = baseUrl + "/"
	}
	w.Header().Set("Location", referer)
	w.WriteHeader(http.StatusFound)
}

// chooseAd queries for advertisements available and randomly chooses one, if
// available. It ignores the error retrieving the ad since it is not critical.
func (fe *frontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *pb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.WithField("error", err).Warn("failed to retrieve ads")
		return nil
	}
	return ads[rand.Intn(len(ads))]
}

func renderHTTPError(log logrus.FieldLogger, r *http.Request, w http.ResponseWriter, err error, code int) {
	log.WithField("error", err).Error("request error")
	errMsg := fmt.Sprintf("%+v", err)

	w.WriteHeader(code)

	if templateErr := templates.ExecuteTemplate(w, "error", injectCommonTemplateData(r, map[string]interface{}{
		"error":       errMsg,
		"status_code": code,
		"status":      http.StatusText(code),
	})); templateErr != nil {
		log.Println(templateErr)
	}
}

func injectCommonTemplateData(r *http.Request, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"session_id":        sessionID(r),
		"request_id":        r.Context().Value(ctxKeyRequestID{}),
		"user_currency":     currentCurrency(r),
		"platform_css":      plat.css,
		"platform_name":     plat.provider,
		"is_cymbal_brand":   isCymbalBrand,
		"assistant_enabled": assistantEnabled,
		"deploymentDetails": deploymentDetailsMap,
		"frontendMessage":   frontendMessage,
		"currentYear":       time.Now().Year(),
		"baseUrl":           baseUrl,
	}

	for k, v := range payload {
		data[k] = v
	}

	return data
}

func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func cartIDs(c []*pb.CartItem) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.GetProductId()
	}
	return out
}

// get total # of items in cart
func cartSize(c []*pb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

func renderMoney(money pb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

func renderCurrencyLogo(currencyCode string) string {
	logos := map[string]string{
		"USD": "$",
		"CAD": "$",
		"JPY": "¥",
		"EUR": "€",
		"TRY": "₺",
		"GBP": "£",
	}

	logo := "$" //default
	if val, ok := logos[currencyCode]; ok {
		logo = val
	}
	return logo
}

func stringinSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
