package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	proxymancmd "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	hyaccount "github.com/xtls/xray-core/proxy/hysteria/account"
	ss "github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserHotReloader struct {
	address string
	mu      sync.Mutex
	current map[string]map[string]RuntimeUser
}

func NewUserHotReloader(apiAddress string) *UserHotReloader {
	return &UserHotReloader{
		address: apiAddress,
		current: map[string]map[string]RuntimeUser{},
	}
}

func (r *UserHotReloader) SetSnapshot(inbounds []InboundUsers) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = usersByInbound(inbounds)
}

func (r *UserHotReloader) Sync(ctx context.Context, desired []InboundUsers) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := usersByInbound(desired)
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(callCtx, r.address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := proxymancmd.NewHandlerServiceClient(conn)
	for _, inbound := range desired {
		currentUsers := r.current[inbound.Tag]
		desiredUsers := next[inbound.Tag]
		for email, currentUser := range currentUsers {
			if desiredUser, ok := desiredUsers[email]; !ok || desiredUser.Hash != currentUser.Hash {
				if err := removeInboundUser(callCtx, client, inbound.Tag, email); err != nil {
					return err
				}
			}
		}
		for email, desiredUser := range desiredUsers {
			if currentUser, ok := currentUsers[email]; ok && currentUser.Hash == desiredUser.Hash {
				continue
			}
			user, err := buildProtocolUser(inbound.Protocol, desiredUser)
			if err != nil {
				return err
			}
			if err := addInboundUser(callCtx, client, inbound.Tag, user); err != nil {
				return err
			}
		}
	}
	for tag, currentUsers := range r.current {
		if _, ok := next[tag]; ok {
			continue
		}
		for email := range currentUsers {
			if err := removeInboundUser(callCtx, client, tag, email); err != nil {
				return err
			}
		}
	}
	r.current = next
	return nil
}

func addInboundUser(ctx context.Context, client proxymancmd.HandlerServiceClient, tag string, user *protocol.User) error {
	_, err := client.AlterInbound(ctx, &proxymancmd.AlterInboundRequest{
		Tag:       tag,
		Operation: serial.ToTypedMessage(&proxymancmd.AddUserOperation{User: user}),
	})
	return err
}

func removeInboundUser(ctx context.Context, client proxymancmd.HandlerServiceClient, tag string, email string) error {
	_, err := client.AlterInbound(ctx, &proxymancmd.AlterInboundRequest{
		Tag:       tag,
		Operation: serial.ToTypedMessage(&proxymancmd.RemoveUserOperation{Email: email}),
	})
	return err
}

func usersByInbound(inbounds []InboundUsers) map[string]map[string]RuntimeUser {
	result := make(map[string]map[string]RuntimeUser, len(inbounds))
	for _, inbound := range inbounds {
		users := make(map[string]RuntimeUser, len(inbound.Users))
		for _, user := range inbound.Users {
			users[user.Email] = user
		}
		result[inbound.Tag] = users
	}
	return result
}

func buildProtocolUser(protocolName string, runtimeUser RuntimeUser) (*protocol.User, error) {
	user := &protocol.User{
		Email: runtimeUser.Email,
		Level: uint32Field(runtimeUser.Fields, "level"),
	}
	switch strings.ToLower(protocolName) {
	case "vless":
		id := stringField(runtimeUser.Fields, "id")
		if id == "" {
			return nil, fmt.Errorf("vless user %s missing id", runtimeUser.Email)
		}
		user.Account = serial.ToTypedMessage(&vless.Account{
			Id:   id,
			Flow: stringField(runtimeUser.Fields, "flow"),
		})
	case "vmess":
		id := stringField(runtimeUser.Fields, "id")
		if id == "" {
			return nil, fmt.Errorf("vmess user %s missing id", runtimeUser.Email)
		}
		user.Account = serial.ToTypedMessage(&vmess.Account{
			Id: id,
			SecuritySettings: &protocol.SecurityConfig{
				Type: vmessSecurityType(stringField(runtimeUser.Fields, "security")),
			},
		})
	case "trojan":
		password := stringField(runtimeUser.Fields, "password")
		if password == "" {
			return nil, fmt.Errorf("trojan user %s missing password", runtimeUser.Email)
		}
		user.Account = serial.ToTypedMessage(&trojan.Account{Password: password})
	case "shadowsocks":
		password := stringField(runtimeUser.Fields, "password")
		if password == "" {
			return nil, fmt.Errorf("shadowsocks user %s missing password", runtimeUser.Email)
		}
		user.Account = serial.ToTypedMessage(&ss.Account{
			Password:   password,
			CipherType: shadowsocksCipherType(stringField(runtimeUser.Fields, "method")),
		})
	case "hysteria":
		auth := stringField(runtimeUser.Fields, "auth")
		if auth == "" {
			return nil, fmt.Errorf("hysteria user %s missing auth", runtimeUser.Email)
		}
		user.Account = serial.ToTypedMessage(&hyaccount.Account{Auth: auth})
	default:
		return nil, fmt.Errorf("protocol %s does not support hot user reload", protocolName)
	}
	return user, nil
}

func stringField(fields map[string]interface{}, key string) string {
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func uint32Field(fields map[string]interface{}, key string) uint32 {
	switch value := fields[key].(type) {
	case int:
		if value > 0 {
			return uint32(value)
		}
	case int64:
		if value > 0 {
			return uint32(value)
		}
	case float64:
		if value > 0 {
			return uint32(value)
		}
	case json.Number:
		if parsed, err := strconv.ParseUint(string(value), 10, 32); err == nil {
			return uint32(parsed)
		}
	case string:
		if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
			return uint32(parsed)
		}
	}
	return 0
}

func vmessSecurityType(value string) protocol.SecurityType {
	switch normalizeToken(value) {
	case "", "auto":
		return protocol.SecurityType_AUTO
	case "aes128gcm":
		return protocol.SecurityType_AES128_GCM
	case "chacha20poly1305":
		return protocol.SecurityType_CHACHA20_POLY1305
	case "none":
		return protocol.SecurityType_NONE
	case "zero":
		return protocol.SecurityType_ZERO
	default:
		return protocol.SecurityType_AUTO
	}
}

func shadowsocksCipherType(value string) ss.CipherType {
	switch normalizeToken(value) {
	case "aes256gcm":
		return ss.CipherType_AES_256_GCM
	case "chacha20poly1305":
		return ss.CipherType_CHACHA20_POLY1305
	case "xchacha20poly1305":
		return ss.CipherType_XCHACHA20_POLY1305
	case "none":
		return ss.CipherType_NONE
	case "", "aes128gcm":
		return ss.CipherType_AES_128_GCM
	default:
		return ss.CipherType_AES_128_GCM
	}
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}
