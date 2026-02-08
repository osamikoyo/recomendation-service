package server

import (
	"context"
	"fmt"
	"time"

	"github.com/osamikoyo/recomendation-service/api/v1/gen"
	"github.com/osamikoyo/recomendation-service/core"
	"github.com/osamikoyo/recomendation-service/logger"
	"github.com/osamikoyo/recomendation-service/metrics"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	gen.UnimplementedRecomendationServiceServer
	logger *logger.Logger
	core   *core.Core
}

func NewServer(core *core.Core, logger *logger.Logger) *Server {
	return &Server{
		core:   core,
		logger: logger,
	}
}

func (s *Server) GetManyRecs(ctx context.Context, req *gen.GetManyRecsRequest) (*gen.GetManyRecsResponse, error) {
	metrics.RequestTotal.WithLabelValues("GetManyRecs").Inc()
	then := time.Now()

	var (
		recs []string
		err  error
	)

	if req.Number == nil {
		recs, err = s.core.GetOrderedRecsForRID(req.LastRid)
	} else {
		recs, err = s.core.GetOrderedRecsForRID(req.LastRid, int(*req.Number))
	}

	if err != nil {
		s.logger.Error("failed get many recs",
			zap.String("rid", req.LastRid),
			zap.Error(err))

		metrics.RequestDuration.WithLabelValues("GetManyRecs").Observe(time.Since(then).Seconds())

		return nil, fmt.Errorf("internal error: %w", err)
	}

	return &gen.GetManyRecsResponse{
		Themes: recs,
	}, nil
}

func (s *Server) GetOneRec(ctx context.Context, req *gen.GetOneRecRequest) (*gen.GetOneRecResponse, error) {
	metrics.RequestTotal.WithLabelValues("GetOneRec").Inc()
	then := time.Now()

	rec, err := s.core.GetBestOneForRID(req.LastRid)
	if err != nil {
		s.logger.Error("failed get one rec",
			zap.String("rid", req.LastRid),
			zap.Error(err))

		return nil, fmt.Errorf("internal error: %w", err)
	}

	metrics.RequestDuration.WithLabelValues("GetOneRec").Observe(time.Since(then).Seconds())

	return &gen.GetOneRecResponse{
		Theme: rec,
	}, nil
}

func (s *Server) RouteAction(ctx context.Context, req *gen.RouteActionRequest) (*emptypb.Empty, error) {
	metrics.RequestTotal.WithLabelValues("RouteAction").Inc()
	then := time.Now()

	if err := s.core.RouteAction(req.LastRid, req.CurrentRid); err != nil {
		s.logger.Error("failed route action",
			zap.String("last_rid", req.LastRid),
			zap.String("current_rid", req.CurrentRid),
			zap.Error(err))

		return &emptypb.Empty{}, fmt.Errorf("internal error: %w", err)
	}

	metrics.RequestDuration.WithLabelValues("RouteAction").Observe(time.Since(then).Seconds())

	return &emptypb.Empty{}, nil
}
