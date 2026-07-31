package adapter

import (
	"github.com/neibniz/alpha/server/internal/model/node"
)

type Adapter struct {
	Type           string            // 协议类型
	SiteName       string            // 站点名称
	Servers        []*node.Node      // 服务器列表
	ExtraProxies   []Proxy           // 外部生成的代理列表
	UserInfo       User              // 用户信息
	ClientTemplate string            // 客户端配置模板
	OutputFormat   string            // 输出格式，默认是 base64
	SubscribeName  string            // 订阅名称
	Params         map[string]string // 其他参数
}

type Option func(*Adapter)

func WithParams(params map[string]string) Option {
	return func(opts *Adapter) {
		opts.Params = params
	}
}

// WithServers 设置服务器列表
func WithServers(servers []*node.Node) Option {
	return func(opts *Adapter) {
		opts.Servers = servers
	}
}

func WithProxies(proxies []Proxy) Option {
	return func(opts *Adapter) {
		opts.ExtraProxies = proxies
	}
}

// WithUserInfo 设置用户信息
func WithUserInfo(user User) Option {
	return func(opts *Adapter) {
		opts.UserInfo = user
	}
}

// WithOutputFormat 设置输出格式
func WithOutputFormat(format string) Option {
	return func(opts *Adapter) {
		opts.OutputFormat = format
	}
}

// WithSiteName 设置站点名称
func WithSiteName(name string) Option {
	return func(opts *Adapter) {
		opts.SiteName = name
	}
}

// WithSubscribeName 设置订阅名称
func WithSubscribeName(name string) Option {
	return func(opts *Adapter) {
		opts.SubscribeName = name
	}
}

func NewAdapter(tpl string, opts ...Option) *Adapter {
	adapter := &Adapter{
		Servers:        []*node.Node{},
		UserInfo:       User{},
		ClientTemplate: tpl,
		OutputFormat:   "base64", // 默认输出格式
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

func (adapter *Adapter) Client() (*Client, error) {
	client := &Client{
		SiteName:       adapter.SiteName,
		SubscribeName:  adapter.SubscribeName,
		ClientTemplate: adapter.ClientTemplate,
		OutputFormat:   adapter.OutputFormat,
		Proxies:        []Proxy{},
		UserInfo:       adapter.UserInfo,
		Params:         adapter.Params,
	}

	proxies, err := adapter.Proxies(adapter.Servers)
	if err != nil {
		return nil, err
	}
	proxies = append(proxies, adapter.ExtraProxies...)
	client.Proxies = proxies
	return client, nil
}

func (adapter *Adapter) Proxies(servers []*node.Node) ([]Proxy, error) {
	return nil, nil
}
