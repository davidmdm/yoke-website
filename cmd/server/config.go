package main

import (
	"os"
	"time"

	"github.com/davidmdm/conf"
)

type Config struct {
	Port        string
	ReadTimeout time.Duration
	GracePeriod time.Duration
}

func GetConfig() (cfg Config) {
	parser := conf.MakeParser(conf.CommandLineArgs(), os.LookupEnv)

	conf.Var(parser, &cfg.Port, "PORT", conf.Default(":8080"))
	conf.Var(parser, &cfg.ReadTimeout, "READ_TIMEOUT", conf.Default(5*time.Second))
	conf.Var(parser, &cfg.GracePeriod, "GRACE_PERIOD", conf.Default(5*time.Second))

	parser.MustParse()

	if len(cfg.Port) > 0 && cfg.Port[0] != ':' {
		cfg.Port = ":" + cfg.Port
	}

	return
}
