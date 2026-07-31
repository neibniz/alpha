package orm

import "net/url"

func ParseDSN(dsn string) *Config {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil
	}
	password, _ := u.User.Password()
	return &Config{
		Addr:          u.Host,
		Dbname:        u.Path[1:],
		Username:      u.User.Username(),
		Password:      password,
		Config:        u.RawQuery,
		MaxIdleConns:  10,
		MaxOpenConns:  10,
		SlowThreshold: 1000,
	}
}
