package internal

import (
	"context"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler(ctx context.Context, _ *any) (*HealthResponse, error) {
	getLogger(ctx).Info("Received health check")
	return &HealthResponse{Status: "healthy"}, nil
}
