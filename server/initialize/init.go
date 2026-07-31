package initialize

import (
	"github.com/neibniz/alpha/server/internal/svc"
)

func StartInitSystemConfig(svc *svc.ServiceContext) {
	Migrate(svc)
	Site(svc)
	Node(svc)
	Email(svc)
	Invite(svc)
	Verify(svc)
	Subscribe(svc)
	Register(svc)
	Currency(svc)
	if !svc.Config.Debug {
		Telegram(svc)
	}

}
