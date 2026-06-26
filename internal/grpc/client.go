package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/webhook-platform/internal/grpc/pb"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.WebhookInternalClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewWebhookInternalClient(conn),
	}, nil
}

func (c *Client) GetEndpoint(ctx context.Context, endpointID string) (*pb.GetEndpointResponse, error) {
	resp, err := c.client.GetEndpoint(ctx, &pb.GetEndpointRequest{EndpointId: endpointID})
	if err != nil {
		return nil, fmt.Errorf("grpc get endpoint: %w", err)
	}
	return resp, nil
}

func (c *Client) GetCircuitBreakerState(ctx context.Context, endpointID string) (string, error) {
	resp, err := c.client.GetCircuitBreakerState(ctx, &pb.GetCircuitBreakerStateRequest{EndpointId: endpointID})
	if err != nil {
		return "", fmt.Errorf("grpc get circuit breaker state: %w", err)
	}
	return resp.State, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
