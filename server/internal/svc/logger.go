package svc

import (
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/logger"
)

func NewLogger(c config.Config) *logger.Logger {
	//log := logger.New(c.Logger)
	//// replace the default logger
	//logger = log
	//return log
	return nil
}
