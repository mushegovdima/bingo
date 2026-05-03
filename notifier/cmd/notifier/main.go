package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notifier/internal/config"
	"notifier/internal/db/repository"
	grpchandler "notifier/internal/grpc"
	pb "notifier/internal/pb/notification/v1"
	"notifier/internal/sender"
	"notifier/internal/worker"

	"buf.build/go/protovalidate"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func main() {
	env := flag.String("env", "prod", "environment (dev|prod)")
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadConfig(*env, *configPath)
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DBConnectionString)))
	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	defer bunDB.Close()

	if err := bunDB.PingContext(context.Background()); err != nil {
		logger.Error("failed to connect to db", slog.Any("error", err))
		os.Exit(1)
	}

	repo := repository.NewNotificationRepository(bunDB)
	handler := grpchandler.New(repo, logger)

	tg, err := sender.NewSender(cfg)
	if err != nil {
		logger.Error("failed to init sender", slog.Any("error", err))
		os.Exit(1)
	}
	workerCfg := worker.Config{
		BatchSize:      cfg.WorkerBatchSize,
		ReservationTTL: time.Duration(cfg.WorkerReservationTTLSec) * time.Second,
	}
	w := worker.New(repo, tg, time.Duration(cfg.WorkerIntervalSec)*time.Second, workerCfg, logger)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		logger.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	validator, err := protovalidate.New()
	if err != nil {
		logger.Error("failed to create validator", slog.Any("error", err))
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if msg, ok := req.(proto.Message); ok {
				if err := validator.Validate(msg); err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}
			}
			return handler(ctx, req)
		}),
	)
	pb.RegisterNotificationServiceServer(grpcSrv, handler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go w.Run(ctx)

	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		grpcSrv.GracefulStop()
	}()

	logger.Info("gRPC server starting", slog.String("addr", cfg.GRPCPort))
	if err := grpcSrv.Serve(lis); err != nil {
		logger.Error("gRPC server error", slog.Any("error", err))
	}
}
