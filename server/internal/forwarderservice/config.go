package forwarderservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/neibniz/alpha/server/internal/forwarderapi"
	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
)

func BuildConfig(ctx context.Context, svcCtx *svc.ServiceContext, instanceID, publicIP string) (*forwarderapi.ConfigResponse, error) {
	logic := server.NewQueryServerForwardTableLogic(ctx, svcCtx)
	table, err := logic.QueryServerForwardTable(&types.QueryServerForwardTableRequest{IP: publicIP})
	if err != nil {
		return nil, err
	}
	rules := make([]forwarderapi.Rule, 0, len(table.Items))
	for _, item := range table.Items {
		if item.LocalPort == 0 || item.LocalPort > 65535 || item.TargetPort == 0 || item.TargetPort > 65535 {
			continue
		}
		rules = append(rules, forwarderapi.Rule{
			RuleID:     ruleID(item),
			ListenPort: uint16(item.LocalPort),
			TargetHost: item.TargetHost,
			TargetPort: uint16(item.TargetPort),
			NodeID:     item.NodeID,
			NodeName:   item.NodeName,
			ServerID:   item.ServerID,
			ServerName: item.ServerName,
			Inbound:    item.Inbound,
		})
	}
	canonical, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	hash := hex.EncodeToString(sum[:])
	return &forwarderapi.ConfigResponse{
		InstanceID: instanceID,
		PublicIP:   publicIP,
		Version:    "v1-" + hash[:16],
		Hash:       hash,
		Rules:      rules,
		Warnings:   table.Warnings,
	}, nil
}

func ruleID(item types.ServerForwardItem) string {
	raw := fmt.Sprintf("%d:%d:%s:%d:%d:%s", item.NodeID, item.LocalPort, item.TargetHost, item.TargetPort, item.ServerID, item.Inbound)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:12])
}
