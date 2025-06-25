package main

import (
    "context"
    "log"
    "net"
    "os"
    "sync"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
    
    // 修改为正确的模块路径
    "github.com/GoogleCloudPlatform/microservices-demo/src/ratingservice/genproto"
)

type ratingServiceServer struct {
    genproto.UnimplementedRatingServiceServer
    ratings map[string][]*genproto.Rating
    mu      sync.RWMutex
}

func (s *ratingServiceServer) SubmitRating(ctx context.Context, req *genproto.SubmitRatingRequest) (*genproto.SubmitRatingResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.ratings == nil {
        s.ratings = make(map[string][]*genproto.Rating)
    }

    rating := &genproto.Rating{
        ProductId: req.ProductId,
        UserId:    req.UserId,
        Score:     req.Score,
        Comment:   req.Comment,
        Timestamp: time.Now().Unix(),
    }

    s.ratings[req.ProductId] = append(s.ratings[req.ProductId], rating)

    log.Printf("Rating submitted: ProductID=%s, UserID=%s, Score=%d, Comment=%s", 
        req.ProductId, req.UserId, req.Score, req.Comment)

    return &genproto.SubmitRatingResponse{
        Success: true,
        Message: "Rating submitted successfully",
    }, nil
}

func (s *ratingServiceServer) GetProductRating(ctx context.Context, req *genproto.GetProductRatingRequest) (*genproto.GetProductRatingResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    ratings := s.ratings[req.ProductId]
    if len(ratings) == 0 {
        log.Printf("No ratings found for product: %s", req.ProductId)
        return &genproto.GetProductRatingResponse{
            ProductId:    req.ProductId,
            AverageScore: 0,
            TotalRatings: 0,
            Ratings:      []*genproto.Rating{},
        }, nil
    }

    var totalScore int32
    for _, rating := range ratings {
        totalScore += rating.Score
    }
    averageScore := float32(totalScore) / float32(len(ratings))

    log.Printf("Retrieved ratings for product %s: %d ratings, average score: %.2f", 
        req.ProductId, len(ratings), averageScore)

    return &genproto.GetProductRatingResponse{
        ProductId:    req.ProductId,
        AverageScore: averageScore,
        TotalRatings: int32(len(ratings)),
        Ratings:      ratings,
    }, nil
}

func (s *ratingServiceServer) GetUserRating(ctx context.Context, req *genproto.GetUserRatingRequest) (*genproto.GetUserRatingResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    ratings := s.ratings[req.ProductId]
    for _, rating := range ratings {
        if rating.UserId == req.UserId {
            return &genproto.GetUserRatingResponse{
                Rating:    rating,
                HasRating: true,
            }, nil
        }
    }

    return &genproto.GetUserRatingResponse{
        Rating:    nil,
        HasRating: false,
    }, nil
}

func main() {
    // 从环境变量获取端口，默认使用 8081
    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }

    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("Failed to listen on port %s: %v", port, err)
    }

    server := grpc.NewServer()
    
    // 注册评分服务
    genproto.RegisterRatingServiceServer(server, &ratingServiceServer{
        ratings: make(map[string][]*genproto.Rating),
    })
    
    // 使用标准健康检查服务
    healthServer := health.NewServer()
    healthServer.SetServingStatus("hipstershop.RatingService", grpc_health_v1.HealthCheckResponse_SERVING)
    grpc_health_v1.RegisterHealthServer(server, healthServer)

    log.Printf("Rating service listening on port %s", port)
    if err := server.Serve(listener); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}