package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/maxliu9403/common/logger"
	"io/ioutil"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
	Endpoints    string `yaml:"endpoints" env:"ETCD_ENDPOINT"`
	DialTimeout  int    `yaml:"dial_timeout"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	CAFilePath   string `yaml:"ca_file_path"`
	CertFilePath string `yaml:"cert_file_path"`
	KeyFilePath  string `yaml:"key_file_path"`
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
