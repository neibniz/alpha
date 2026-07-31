package adapter

import (
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
)

func TestAdapterProxy(t *testing.T) {

	servers := getServers()
	if len(servers) == 0 {
		t.Fatal("no servers found")
	}
	a := NewAdapter("", WithServers(servers))
	proxies, err := a.Proxies(servers)
	if err != nil {
		t.Fatalf("failed to adapt servers: %v", err)
	}
	for _, proxy := range proxies {
		t.Logf("[测试] 适配服务器成功: %+v", proxy)
	}
}

func getServers() []*node.Node {
	server := &node.Server{
		Id:      1,
		Name:    "test-server",
		Address: "example.com",
	}
	return []*node.Node{{
		Id:       1,
		Name:     "test-node",
		Tags:     "test",
		Port:     443,
		Address:  "example.com",
		ServerId: 1,
		Server:   server,
		Protocol: "shadowsocks",
		Sort:     1,
	}}
}
