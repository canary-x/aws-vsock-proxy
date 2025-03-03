package internal

import (
	"context"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func getLogger(_ context.Context) *zap.Logger {
	return logger
}
