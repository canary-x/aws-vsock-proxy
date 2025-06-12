package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

type UploadFileRequest struct {
	Bucket      string `json:"bucket"`      // mandatory bucket name
	Key         string `json:"key"`         // mandatory file key (absolute path)
	ContentType string `json:"contentType"` // optional
	Data        string `json:"data"`        // data encoded as b64
}

type UploadFileResponse struct{}

func S3PutHandler(s3Client *s3.Client) HandlerFunc[UploadFileRequest, UploadFileResponse] {
	return func(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
		if req.Bucket == "" {
			return nil, BadRequest("bucket is required")
		}
		if req.Key == "" {
			return nil, BadRequest("key is required")
		}
		if req.Data == "" {
			return nil, BadRequest("data is required")
		}

		data, err := base64.StdEncoding.DecodeString(req.Data)
		if err != nil {
			return nil, BadRequest("invalid base64 data: " + err.Error())
		}

		getLogger(ctx).Info("Uploading file to S3",
			zap.String("bucket", req.Bucket),
			zap.String("key", req.Key),
			zap.String("content_type", req.ContentType),
			zap.Int("data_length", len(data)),
		)

		input := &s3.PutObjectInput{
			Bucket: aws.String(req.Bucket),
			Key:    aws.String(req.Key),
			Body:   bytes.NewReader(data),
		}
		if req.ContentType != "" {
			input.ContentType = aws.String(req.ContentType)
		}

		_, err = s3Client.PutObject(ctx, input)
		return &UploadFileResponse{}, errors.Wrapf(err, "uploading file to s3://%s/%s", req.Bucket, req.Key)
	}
}
