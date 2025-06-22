package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/maxliu9403/common/logger"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
	Endpoints    string `yaml:"endpoints" env:"ETCD_ENDPOINT" env-description:"address of etcd cluster"`
	DialTimeout  int    `yaml:"dial_timeout" env:"DIAL_TIMEOUT" env-default:"5" env-description:"is the timeout for failing to establish a connection"`
	Username     string `yaml:"username" env:"USER_NAME" env-description:"username of etcd cluster"`
	Password     string `yaml:"password" env:"PASS_WORD" env-description:"password of etcd cluster"`
	CAFilePath   string `yaml:"ca_file_path" env:"CA_FILE_PATH"`
	CertFilePath string `yaml:"cert_file_path" env:"CERT_FILE_PATH"`
	KeyFilePath  string `yaml:"key_file_path" env:"KEY_FILE_PATH"`
}

func (c *Config) Init(ctx context.Context) error {
	addr := strings.Split(c.Endpoints, ",")
	if len(addr) == 0 || addr[0] == "" {
		return fmt.Errorf("no endpoints specified: [%+v]", c)
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 5
	}

	etcdCli := &CliConfig{
		ctx: ctx,
	}
	etcdCli.etcdConfig = clientv3.Config{
		Endpoints:   addr,
		DialTimeout: time.Duration(c.DialTimeout) * time.Second,
		Username:    c.Username,
		Password:    c.Password,
		Context:     ctx,
	}
	etcdCli.etcdConfig.Logger = logger.Default().Desugar()
	if c.CAFilePath != "" && c.CertFilePath != "" && c.KeyFilePath != "" {
		_tlsConfig, err := createTLS(c.CAFilePath, c.CertFilePath, c.KeyFilePath)
		if err != nil {
			return err
		}
		etcdCli.etcdConfig.TLS = _tlsConfig
	}

	if _defaultCliCfg == nil {
		_defaultCliCfg = etcdCli
	}

	return nil
}

func createTLS(ca, cert, key string) (*tls.Config, error) {
	certed, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	caData, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)

	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certed},
		RootCAs:      pool,
	}

	return _tlsConfig, nil
}
