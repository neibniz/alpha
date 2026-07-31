package xray

import (
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

func InjectUsers(raw []byte, users []panel.XrayUser) ([]byte, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	inbounds := asSlice(cfg["inbounds"])
	for _, item := range inbounds {
		inbound := asMap(item)
		protocol := strings.ToLower(stringValue(inbound["protocol"]))
		tag := stringValue(inbound["tag"])
		if tag == "api" {
			continue
		}
		clients := clientsForInbound(protocol, tag, users)
		if clients == nil {
			continue
		}
		settings := asMap(inbound["settings"])
		userListKey := userListField(protocol)
		defaultClient := configuredUserDefaults(settings, userListKey)
		settings[userListKey] = mergeClients(defaultClient, clients)
		if userListKey == "clients" {
			delete(settings, "users")
		} else {
			delete(settings, "clients")
		}
		inbound["settings"] = settings
	}
	cfg["inbounds"] = inbounds
	return json.Marshal(cfg)
}

func userListField(protocol string) string {
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks":
		return "clients"
	default:
		return "users"
	}
}

func configuredUserDefaults(settings map[string]interface{}, userListKey string) map[string]interface{} {
	value, ok := settings[userListKey]
	if !ok {
		if userListKey == "clients" {
			value = settings["users"]
		} else {
			value = settings["clients"]
		}
	}
	return firstClientDefaults(value)
}

func clientsForInbound(protocol string, tag string, users []panel.XrayUser) []map[string]interface{} {
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks", "hysteria":
	default:
		return nil
	}
	clients := make([]map[string]interface{}, 0)
	for _, user := range users {
		if !user.Enabled || !userMatchesInbound(user, tag) {
			continue
		}
		client := map[string]interface{}{
			"email": user.Email,
			"level": user.Level,
		}
		switch protocol {
		case "vless", "vmess":
			client["id"] = user.UUID
		case "trojan", "shadowsocks":
			client["password"] = choosePassword(user)
		case "hysteria":
			client["auth"] = choosePassword(user)
		}
		clients = append(clients, client)
	}
	return clients
}

func firstClientDefaults(value interface{}) map[string]interface{} {
	clients := asSlice(value)
	if len(clients) == 0 {
		return map[string]interface{}{}
	}
	defaults := asMap(clients[0])
	result := make(map[string]interface{}, len(defaults))
	for key, value := range defaults {
		result[key] = value
	}
	return result
}

func mergeClients(defaultClient map[string]interface{}, identities []map[string]interface{}) []interface{} {
	clients := make([]interface{}, 0, len(identities))
	for _, identity := range identities {
		clients = append(clients, mergeClient(defaultClient, identity))
	}
	return clients
}

func mergeClientsToMaps(defaultClient map[string]interface{}, identities []map[string]interface{}) []map[string]interface{} {
	clients := make([]map[string]interface{}, 0, len(identities))
	for _, identity := range identities {
		clients = append(clients, mergeClient(defaultClient, identity))
	}
	return clients
}

func mergeClient(defaultClient map[string]interface{}, identity map[string]interface{}) map[string]interface{} {
	client := make(map[string]interface{}, len(defaultClient)+len(identity))
	for key, value := range defaultClient {
		client[key] = value
	}
	for key, value := range identity {
		client[key] = value
	}
	return client
}

func userMatchesInbound(user panel.XrayUser, tag string) bool {
	if len(user.Inbounds) == 0 || tag == "" {
		return true
	}
	for _, inbound := range user.Inbounds {
		if inbound == tag {
			return true
		}
	}
	return false
}

func choosePassword(user panel.XrayUser) string {
	if user.Password != "" {
		return user.Password
	}
	return user.UUID
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}
