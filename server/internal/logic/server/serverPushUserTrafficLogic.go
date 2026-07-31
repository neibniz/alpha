package server

import (
	"context"
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	task "github.com/neibniz/alpha/server/queue/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
)

const (
	maxTrafficReportsPerRequest = 120
	maxTrafficRowsPerRequest    = 20_000
	maxTrafficClockSkew         = 5 * time.Minute
)

//goland:noinspection GoNameStartsWithPackageName
type ServerPushUserTrafficLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewServerPushUserTrafficLogic Push user Traffic
func NewServerPushUserTrafficLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ServerPushUserTrafficLogic {
	return &ServerPushUserTrafficLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ServerPushUserTrafficLogic) ServerPushUserTraffic(req *types.ServerPushUserTrafficRequest) error {
	reports, err := normalizeTrafficReports(req, time.Now())
	if err != nil {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error())
	}
	if len(reports) == 0 {
		return nil
	}

	exists, err := l.svcCtx.NodeModel.ServerExists(l.ctx, req.ServerId)
	if err != nil || !exists {
		l.Errorw("[ServerPushUserTraffic] FindOne error", logger.Field("error", err))
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "server not found")
	}

	// Create traffic task
	var request agentprotocol.TrafficTask
	request.ServerID = req.ServerId
	request.Protocol = req.Protocol
	request.Reports = reports

	// Push traffic task
	val, err := agentprotocol.MarshalTrafficTask(request)
	if err != nil {
		l.Errorw("[ServerPushUserTraffic] Marshal traffic task error", logger.Field("error", err.Error()), logger.Field("traffic_count", len(req.Traffic)))
		return errors.Wrap(err, "marshal traffic task")
	}
	t := asynq.NewTask(task.ForthwithTrafficStatisticsV2, val, asynq.MaxRetry(25), asynq.Unique(5*time.Minute))
	info, err := l.svcCtx.Queue.EnqueueContext(l.ctx, t)
	if err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
		l.Errorw("[ServerPushUserTraffic] Push traffic task error",
			logger.Field("error", err.Error()),
			logger.Field("task_type", t.Type()),
			logger.Field("report_count", len(request.Reports)),
			logger.Field("payload_bytes", len(val)),
		)
		return errors.Wrap(err, "push traffic task")
	} else {
		l.Infow("[ServerPushUserTraffic] Push traffic task success",
			logger.Field("task_id", info.ID),
			logger.Field("task_type", t.Type()),
			logger.Field("report_count", len(request.Reports)),
			logger.Field("payload_bytes", len(val)),
		)
	}

	return nil
}

func normalizeTrafficReports(req *types.ServerPushUserTrafficRequest, now time.Time) ([]types.TrafficReport, error) {
	if req == nil {
		return nil, errors.New("traffic request is nil")
	}
	if len(req.Reports) > maxTrafficReportsPerRequest {
		return nil, fmt.Errorf("traffic report batch exceeds %d reports", maxTrafficReportsPerRequest)
	}
	if len(req.Reports) == 0 {
		if len(req.Traffic) == 0 {
			return nil, nil
		}
		if len(req.Traffic) > maxTrafficRowsPerRequest {
			return nil, fmt.Errorf("traffic report exceeds %d rows", maxTrafficRowsPerRequest)
		}
		return []types.TrafficReport{{
			ReportId:        uuid.NewString(),
			SampledAt:       now.UnixMilli(),
			IntervalSeconds: 30,
			Sequence:        1,
			Traffic:         req.Traffic,
		}}, nil
	}

	totalRows := 0
	reports := make([]types.TrafficReport, 0, len(req.Reports))
	for _, report := range req.Reports {
		reportID, err := uuid.Parse(report.ReportId)
		if err != nil {
			return nil, fmt.Errorf("invalid traffic report id: %w", err)
		}
		if report.SampledAt <= 0 {
			return nil, fmt.Errorf("traffic report sampled_at must be positive")
		}
		sampledAt := time.UnixMilli(report.SampledAt)
		if sampledAt.After(now.Add(maxTrafficClockSkew)) {
			return nil, fmt.Errorf("traffic report sampled_at is in the future")
		}
		if report.IntervalSeconds < 1 || report.IntervalSeconds > 3600 {
			return nil, fmt.Errorf("traffic report interval must be between 1 and 3600 seconds")
		}
		if report.Sequence < 0 {
			return nil, fmt.Errorf("traffic report sequence cannot be negative")
		}
		totalRows += len(report.Traffic)
		if totalRows > maxTrafficRowsPerRequest {
			return nil, fmt.Errorf("traffic report batch exceeds %d rows", maxTrafficRowsPerRequest)
		}
		if len(report.Traffic) == 0 {
			continue
		}
		report.ReportId = reportID.String()
		reports = append(reports, report)
	}
	return reports, nil
}
