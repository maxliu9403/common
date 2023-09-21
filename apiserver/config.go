/*
@Date: 2021/11/10 11:27
@Author: max.liu
@File : config
*/

package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/maxliu9403/common/etcd"
	"github.com/maxliu9403/common/ratelimiter"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/maxliu9403/common/gormdb"
	"github.com/maxliu9403/common/kafka"
	"github.com/maxliu9403/common/logger"
	"github.com/maxliu9403/common/rediscache"
	"github.com/maxliu9403/common/tracer"
	"github.com/spf13/cobra"
)

const (
	RunModeDebug      = "debug"
	RunModeTest       = "test"
	RunModeDev        = "dev"
	RunModeProd       = "prod"
	RunModeProduction = "production"
	RunModeRelease    = "release"
)

type APIConfig struct {
	App         AppConfig                 `yaml:"app"`
	Log         logger.Config             `yaml:"log"`
	MySQL       gormdb.DBConfig           `yaml:"mysql"`
	Redis       rediscache.Config         `yaml:"redis"`
	Kafka       kafka.Config              `yaml:"kafka"`
	Tracer      tracer.Config             `yaml:"tracer"`
	RateLimiter ratelimiter.LimiterConfig `yaml:"ratelimiter"`
	Etcd        etcd.Config               `yaml:"etcd"`
}

type AppConfig struct {
	ServiceName    string `yaml:"service_name" env-default:"gin-project" env-description:"the name of the service"`
	HostIP         string `yaml:"local_ip" env:"HostIP" env-default:"0.0.0.0" env-description:"listening on which IP"`
	APIPort        int    `yaml:"api_port" env:"APIPort" env-default:"8000" env-description:"listening on which port"`
	AdminPort      int    `yaml:"admin_port" env:"AdminPort" env-default:"8001" env-description:"listening on which port of admin service"`
	RunMode        string `yaml:"run_mode" env:"RunMode" env-description:"run mode of the service"`
	RefreshMinutes int    `yaml:"refresh_minutes" env:"RefreshMin" env-default:"5"`
	CertFile       string `yaml:"cert_file" env:"CertFile" env-description:"cert file if server need to use tls"`
	KeyFile        string `yaml:"key_file" env:"KeyFile" env-description:"key file if server need to use tls"`
	Cors           string `yaml:"cors" env:"Cors" env-default:"1" env-description:"cors"`
}

func (c *APIConfig) buildLogger() *logger.DemoLog {
	if c.Log.LogName == "" {
		c.Log.LogName = c.App.ServiceName
	}

	return logger.ConfigureLogger(&logger.Options{Config: c.Log})
}

func (c *APIConfig) String() string {
	configData, err := json.Marshal(c)
	if err != nil {
		fmt.Println(err)
	}

	return string(configData)
}

func (c *APIConfig) initService(ctx context.Context, opts *serverOptions) (err error) {
	// 初始化限流器
	c.RateLimiter.BuildRateLimiter(ctx)
	if c.MySQL.WriteDBHost != "" {
		if opts.tableColumnWithRaw {
			c.MySQL.RawColumn = true
		}

		db, e := c.MySQL.BuildMySQLClient(ctx)
		if e != nil {
			err = e
			return
		}

		if len(opts.migrationList) > 0 {
			err = db.Migration(opts.migrationList...)
			if err != nil {
				return
			}
		}
	}

	if c.Redis.Addr != "" {
		err = c.Redis.NewRedisCli(ctx)
		if err != nil {
			return
		}
	}

	if c.Kafka.Addr != "" {
		_, err = c.Kafka.BuildKafka(ctx)
		if err != nil {
			return
		}
	}

	if c.Etcd.Endpoints != "" {
		err = c.Etcd.Init(ctx)
		if err != nil {
			return
		}
		err = etcd.Default().CreateEtcdV3Client()
		if err != nil {
			return
		}
	}

	return err
}

func NewConfigEnvCommand(c interface{}) *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Prints environment variables.",
		Run: func(*cobra.Command, []string) {
			help, _ := cleanenv.GetDescription(c, nil)
			fmt.Println(help)
		},
	}
}
