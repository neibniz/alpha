package initialize

import (
	"context"
	"encoding/json"
	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/system"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/nodeMultiplier"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func Node(ctx *svc.ServiceContext) {
	logger.Debug("Node config initialization")
	configs, err := ctx.SystemModel.GetNodeConfig(context.Background())
	if err != nil {
		panic(err)
	}
	var nodeConfig config.NodeDBConfig
	tool.SystemConfigSliceReflectToStruct(configs, &nodeConfig)
	runtime := ctx.ConfigSnapshot()
	deploySecret := runtime.Node.NodeDeploySecret
	nodeSecret := runtime.Node.NodeSecret
	c := config.NodeConfig{
		NodeSecret:             nodeSecret,
		NodeDeploySecret:       deploySecret,
		NodePullInterval:       nodeConfig.NodePullInterval,
		NodePushInterval:       nodeConfig.NodePushInterval,
		IPStrategy:             nodeConfig.IPStrategy,
		TrafficReportThreshold: nodeConfig.TrafficReportThreshold,
	}

	ctx.SetNodeConfig(c)
	ctx.SetNodeMultiplierManager(nil)

	// Manager initialization
	if ctx.DB.Model(&system.System{}).Where("key = ?", "NodeMultiplierConfig").Find(&system.System{}).RowsAffected == 0 {
		if err := ctx.DB.Model(&system.System{}).Create(&system.System{
			Key:      "NodeMultiplierConfig",
			Value:    "[]",
			Type:     "string",
			Desc:     "Node Multiplier Config",
			Category: "server",
		}).Error; err != nil {
			logger.Errorf("Create Node Multiplier Config Error: %s", err.Error())
		}
		return
	}

	nodeMultiplierData, err := ctx.SystemModel.FindNodeMultiplierConfig(context.Background())
	if err != nil {
		logger.Error("Get Node Multiplier Config Error: ", logger.Field("error", err.Error()))
		return
	}
	var periods []nodeMultiplier.TimePeriod
	if err := json.Unmarshal([]byte(nodeMultiplierData.Value), &periods); err != nil {
		logger.Error("Unmarshal Node Multiplier Config Error: ", logger.Field("error", err.Error()), logger.Field("value", nodeMultiplierData.Value))
	}
	ctx.SetNodeMultiplierManager(nodeMultiplier.NewNodeMultiplierManager(periods))
}
