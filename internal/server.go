package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Server struct {
	h   http.Handler
	cfg config
	srv *http.Server
}

func NewServer(h http.Handler, cfg config) *Server {
	return &Server{
		h:   h,
		cfg: cfg,
	}
}

func (s *Server) Serve(ln net.Listener) error {
	s.srv = &http.Server{
		Handler:      s.h,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}
	return s.srv.Serve(ln)
}

func (s *Server) Close() error {
	return s.srv.Close()
}

func listen(cfg config, log *zap.Logger) (net.Listener, error) {
	listenTCP := func(port uint32) (net.Listener, error) {
		return net.Listen("tcp", fmt.Sprintf(":%d", port))
	}
	contextID, err := vsock.ContextID()
	if err != nil {
		log.Sugar().Infof("OS does not support vsock (error on getting CID: %v): falling back to regular TCP socket", err)
		return listenTCP(cfg.ServerPort)
	}

	ln, err := vsock.ListenContextID(contextID, cfg.ServerPort, nil)
	if err != nil && strings.Contains(err.Error(), "vsock: not implemented") {
		log.Warn("OS does not support vsock: falling back to regular TCP socket")
		return listenTCP(cfg.ServerPort)
	}
	log.Sugar().Infof("Vsock connected on CID %d", contextID)
	return ln, err
}

func registerRoutes(ctx context.Context, rtr *mux.Router) error {
	log := getLogger(ctx)
	log.Info("Loading AWS config")

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "loading AWS config")
	}
	sm := secretsmanager.NewFromConfig(awsCfg)
	log.Info("AWS config loaded")

	rtr.HandleFunc("/", APIHandler(HealthHandler)).Methods(http.MethodGet)
	rtr.HandleFunc("/secret", APIHandler(GetSecretHandler(sm))).Methods(http.MethodGet)
	return nil
}

type BadRequestError struct {
	Reason string
}

func (b *BadRequestError) Error() string {
	return b.Reason
}

type Response[T any] struct {
	StatusCode int `json:"statusCode"`
	Body       T   `json:"body"`
}

func OkResponse[T any](body T) Response[T] {
	return Response[T]{
		StatusCode: http.StatusOK,
		Body:       body,
	}
}

func BadRequest(reason string) error {
	return &BadRequestError{Reason: reason}
}

type NotFoundError struct {
	Reason string
}

func (b *NotFoundError) Error() string {
	return b.Reason
}

func NotFound(reason string) error {
	return &NotFoundError{Reason: reason}
}

type HandlerFunc[I, O any] func(context.Context, *I) (*O, error)

func APIHandler[I, O any](handler HandlerFunc[I, O]) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := getLogger(ctx)

		handleBadRequest := func(badReqErr *BadRequestError) {
			log.Warn("Invalid request detected", zap.String("reason", badReqErr.Reason))
			resp.WriteHeader(http.StatusBadRequest)
			errResp := Response[string]{http.StatusBadRequest, badReqErr.Reason}
			if err := json.NewEncoder(resp).Encode(errResp); err != nil {
				log.Error("Error encoding response", zap.Error(err))
			}
		}
		handleNotFound := func(notFoundErr *NotFoundError) {
			log.Warn("Not found", zap.String("reason", notFoundErr.Reason))
			resp.WriteHeader(http.StatusNotFound)
			errResp := Response[string]{http.StatusNotFound, notFoundErr.Reason}
			if err := json.NewEncoder(resp).Encode(errResp); err != nil {
				log.Error("Error encoding response", zap.Error(err))
			}
		}
		handleError := func(err error) {
			log.Error("Error handling request", zap.Error(err))
			resp.WriteHeader(http.StatusInternalServerError)
			errResp := Response[string]{http.StatusInternalServerError, "Internal server error"}
			if err := json.NewEncoder(resp).Encode(errResp); err != nil {
				log.Error("Error encoding response", zap.Error(err))
			}
		}

		input := new(I)
		if len(req.URL.Query()) > 0 {
			// decode query parameters into input struct
			if err := schema.NewDecoder().Decode(input, req.URL.Query()); err != nil {
				handleBadRequest(&BadRequestError{Reason: err.Error()})
				return
			}
		} else {
			// decode request body into input struct
			err := json.NewDecoder(req.Body).Decode(input)
			if err != nil && !errors.Is(err, io.EOF) { // empty body produces empty struct
				handleBadRequest(&BadRequestError{Reason: err.Error()})
				return
			}
		}

		output, err := handleWithPanicRecovery(ctx, log, input, handler)
		if err != nil {
			var badReq *BadRequestError
			if errors.As(err, &badReq) {
				handleBadRequest(badReq)
				return
			}
			var notFound *NotFoundError
			if errors.As(err, &notFound) {
				handleNotFound(notFound)
				return
			}
			handleError(err)
			return
		}

		resp.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(resp).Encode(OkResponse(output)); err != nil {
			log.Error("Error encoding bad request response", zap.Error(err))
		}
	}
}

func handleWithPanicRecovery[I, O any](
	ctx context.Context, log *zap.Logger, req *I, handler HandlerFunc[I, O],
) (resp *O, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovering from panic in http handler", zap.Any("error", r))
			err = errors.Errorf("panic: %s", r)
		}
	}()
	return handler(ctx, req)
}
