package xray

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

type InboundUsers struct {
	Tag      string
	Protocol string
	Users    []RuntimeUser
}

type RuntimeUser struct {
	Email  string
	Fields map[string]interface{}
	Hash   string
}

func BuildInboundUsers(raw []byte, users []panel.XrayUser) ([]InboundUsers, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	result := make([]InboundUsers, 0)
	for _, item := range asSlice(cfg["inbounds"]) {
		inbound := asMap(item)
		tag := stringValue(inbound["tag"])
		if tag == "" || tag == "api" {
			continue
		}
		protocol := strings.ToLower(stringValue(inbound["protocol"]))
		identities := clientsForInbound(protocol, tag, users)
		if identities == nil {
			continue
		}
		settings := asMap(inbound["settings"])
		defaultClient := configuredUserDefaults(settings, userListField(protocol))
		clients := mergeClientsToMaps(defaultClient, identities)
		runtimeUsers := make([]RuntimeUser, 0, len(clients))
		for _, client := range clients {
			email := stringValue(client["email"])
			if email == "" {
				continue
			}
			runtimeUsers = append(runtimeUsers, RuntimeUser{
				Email:  email,
				Fields: client,
				Hash:   stableHash(client),
			})
		}
		sort.Slice(runtimeUsers, func(i, j int) bool {
			return runtimeUsers[i].Email < runtimeUsers[j].Email
		})
		result = append(result, InboundUsers{
			Tag:      tag,
			Protocol: protocol,
			Users:    runtimeUsers,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Tag < result[j].Tag
	})
	return result, nil
}

func stableHash(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return hashBytes(data)
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
