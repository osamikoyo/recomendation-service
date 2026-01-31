package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/osamikoyo/recomendation-service/api/v1/gen"
	"github.com/osamikoyo/recomendation-service/config"
	"github.com/osamikoyo/recomendation-service/core"
	"github.com/osamikoyo/recomendation-service/logger"
	"github.com/osamikoyo/recomendation-service/metrics"
	"github.com/osamikoyo/recomendation-service/repository"
	"github.com/osamikoyo/recomendation-service/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type App struct {
	logger          *logger.Logger
	cfg             *config.Config
	server          *grpc.Server
	metricsServer   *http.Server
	driverCloseFunc func(ctx context.Context) error
}

func SetupApp(logger *logger.Logger, cfg *config.Config) (*App, error) {
	logger.Info("setupping app...",
		zap.Any("cfg", cfg))

	DBdriver, err := neo4j.NewDriverWithContext(cfg.DB.Addr, neo4j.BasicAuth(
		cfg.DB.Username, cfg.DB.Password, "",
	))
	if err != nil {
		logger.Error("faield connect to neo4j",
			zap.String("url", cfg.DB.Addr),
			zap.String("username", cfg.DB.Username),
			zap.Error(err))

		return nil, fmt.Errorf("faield connect to db: %w", err)
	}

	repo := repository.NewRepository(DBdriver, logger)

	core := core.NewCore(repo, cfg.Timeout)

	server := server.NewServer(core, logger)

	grpcserv := grpc.NewServer()
	gen.RegisterRecomendationServiceServer(grpcserv, server)

	logger.Info("app was setupped successfully")

	return &App{
		cfg:    cfg,
		logger: logger,
		server: grpcserv,
		metricsServer: &http.Server{
			Addr: cfg.MetricsAddr,
		},
		driverCloseFunc: func(ctx context.Context) error {
			return DBdriver.Close(ctx)
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting app...",
		zap.String("metrics_addr", a.cfg.MetricsAddr),
		zap.String("addr", a.cfg.Addr))

	metrics.InitMetrics()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	lis, err := net.Listen("tcp", a.cfg.Addr)
	if err != nil {
		a.logger.Error("failed listen",
			zap.String("addr", a.cfg.Addr),
			zap.Error(err))

		return fmt.Errorf("faield listen: %w", err)
	}

	errors := make(chan error, 2)

	var wg sync.WaitGroup

	wg.Go(func() {
		if err := a.server.Serve(lis);err != nil{
			errors <- err
		}
	})

	wg.Go(func() {
		if err := a.metricsServer.ListenAndServe();err != nil{
			errors <- err
		}
	})
}

func (a *App) Close(ctx context.Context) error {
	a.logger.Info("closing app...")

	a.server.GracefulStop()

	return a.driverCloseFunc(ctx)
}
