package server

import (
	"context"
	"strings"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	api "modulo.com/proyecto_distribuido/api/v1"
	auth "modulo.com/proyecto_distribuido/auth"
	"modulo.com/proyecto_distribuido/log"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var _ api.LogServer = (*grpcServer)(nil)

// Config contiene la configuración necesaria para el servidor.
type Config struct {
	CommitLog  *log.Log
	Authorizer *auth.Authorizer
}

// grpcServer implementa la interfaz LogServer de gRPC.
type grpcServer struct {
	*api.UnimplementedLogServer
	CommitLog *log.Log
}

func newgrpcServer(config *Config) (*grpcServer, error) {
	srv := &grpcServer{
		UnimplementedLogServer: &api.UnimplementedLogServer{},
		CommitLog:              config.CommitLog,
	}
	return srv, nil
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	logger := zap.L().Named("server")
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(func(duration time.Duration) zapcore.Field {
			return zap.Int64(
				"grpc.time_ns",
				duration.Nanoseconds(),
			)
		},
		),
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	err := view.Register(ocgrpc.DefaultServerViews...)
	if err != nil {
		return nil, err
	}
	halfSampler := trace.ProbabilitySampler(0.5)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: func(p trace.SamplingParameters) trace.SamplingDecision {
			if strings.Contains(p.Name, "Produce") {
				return trace.SamplingDecision{Sample: true}
			}
			return halfSampler(p)
		},
	})
	opts = append(opts, grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(logger, zapOpts...),
			grpc_auth.StreamServerInterceptor(authenticate),
		)), grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_auth.UnaryServerInterceptor(authenticate),
	)))
	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}
func authenticate(ctx context.Context) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find p info",
		).Err()
	}

	if p.AuthInfo == nil {
		return ctx, status.New(
			codes.Unauthenticated,
			"no transport security being used",
		).Err()
	}

	tlsInfo := p.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}
func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}

// Produce permite agregar un registro al log.
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	if req == nil || req.Record == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

// Consume permite leer un registro del log.
func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

// ProduceStream maneja un stream de solicitudes Produce.
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			req, err := stream.Recv()
			if err != nil {
				return err
			}

			res, err := s.Produce(stream.Context(), req)
			if err != nil {
				return err
			}

			if err := stream.Send(res); err != nil {
				return err
			}
		}
	}
}

// ConsumeStream maneja un stream de solicitudes Consume.
func (s *grpcServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	if req == nil {
		return status.Errorf(codes.InvalidArgument, "invalid request")
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}

			if err := stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}
