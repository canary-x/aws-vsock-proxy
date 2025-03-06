package internal

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler(ctx context.Context, _ *any) (*HealthResponse, error) {
	getLogger(ctx).Info("Received health check")
	return &HealthResponse{Status: "healthy"}, nil
}

type GetSecretRequest struct {
	SecretId string `json:"secretId"`
}

type GetSecretResponse struct {
	Value string `json:"value"`
}

func GetSecretHandler(sm *secretsmanager.Client) HandlerFunc[GetSecretRequest, GetSecretResponse] {
	return func(ctx context.Context, req *GetSecretRequest) (*GetSecretResponse, error) {
		if req.SecretId == "" {
			return nil, BadRequest("secretId is required")
		}
		getLogger(ctx).Info("Retrieving secret from AWS", zap.String("secret_id", req.SecretId))

		secret, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &req.SecretId})
		if err != nil {
			if strings.Contains(err.Error(), "Code: 400") {
				return nil, NotFound("secret not found: " + req.SecretId)
			}
			return nil, errors.Wrapf(err, "getting secret %s", req.SecretId)
		}
		if secret.SecretString != nil {
			return &GetSecretResponse{Value: *secret.SecretString}, nil
		}
		if secret.SecretBinary != nil {
			return &GetSecretResponse{Value: string(secret.SecretBinary)}, nil
		}
		return nil, errors.New("aws secret client returned neither string nor binary value")
	}
}
