package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/webhook-platform/internal/domain"
	pb "github.com/webhook-platform/internal/grpc/pb"
	"github.com/webhook-platform/internal/service"
)

type WebhookInternalServer struct {
	pb.UnimplementedWebhookInternalServer
	endpointSvc    service.EndpointService
	circuitBreaker service.CircuitBreakerService
	logger         *slog.Logger
}

func NewWebhookInternalServer(
	endpointSvc service.EndpointService,
	circuitBreaker service.CircuitBreakerService,
	logger *slog.Logger,
) *WebhookInternalServer {
	return &WebhookInternalServer{
		endpointSvc:    endpointSvc,
		circuitBreaker: circuitBreaker,
		logger:         logger,
	}
}

func (s *WebhookInternalServer) GetEndpoint(ctx context.Context, req *pb.GetEndpointRequest) (*pb.GetEndpointResponse, error) {
	if req.EndpointId == "" {
		return nil, status.Error(codes.InvalidArgument, "endpoint_id is required")
	}

	endpoint, err := s.endpointSvc.GetByID(ctx, req.EndpointId)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, "endpoint not found")
		}
		s.logger.Error("grpc get endpoint", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.GetEndpointResponse{
		Id:         endpoint.ID,
		TenantId:   endpoint.TenantID,
		Url:        endpoint.URL,
		Secret:     endpoint.Secret,
		Active:     endpoint.Active,
		EventTypes: endpoint.EventTypes,
	}, nil
}

func (s *WebhookInternalServer) GetCircuitBreakerState(ctx context.Context, req *pb.GetCircuitBreakerStateRequest) (*pb.GetCircuitBreakerStateResponse, error) {
	if req.EndpointId == "" {
		return nil, status.Error(codes.InvalidArgument, "endpoint_id is required")
	}

	state, err := s.circuitBreaker.GetState(ctx, req.EndpointId)
	if err != nil {
		s.logger.Error("grpc get circuit breaker state", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.GetCircuitBreakerStateResponse{State: state}, nil
}

func StartGRPCServer(port string, srv *WebhookInternalServer, logger *slog.Logger) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWebhookInternalServer(grpcServer, srv)
	reflection.Register(grpcServer)

	logger.Info("starting gRPC server", slog.String("port", port))

	return grpcServer.Serve(lis)
}
