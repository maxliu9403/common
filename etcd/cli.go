package etcd

import (
	"context"
	"fmt"
	"github.com/maxliu9403/common/logger"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	_defaultCliCfg *CliConfig
	_defaultCli    *Client
)

type CliConfig struct {
	Endpoints  []string
	etcdConfig clientv3.Config
	ctx        context.Context
}

type Client struct {
	ctx context.Context
	cli *clientv3.Client
}

func Default() *CliConfig {
	return _defaultCliCfg
}

func Cli() *Client {
	if _defaultCli == nil {
		return &Client{}
	}

	return _defaultCli
}

func (c *CliConfig) CreateEtcdV3Client() error {
	if c == nil {
		return fmt.Errorf("etcd config is not initialized yet")
	}
	if _defaultCli != nil {
		return nil
	}

	cli, err := clientv3.New(c.etcdConfig)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	m, err := cli.Cluster.MemberList(c.ctx)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Debugf("etcd cluster member list: %s", m.Members)

	ctx, cancel := context.WithCancel(c.ctx)
	_defaultCli = &Client{ctx: ctx, cli: cli}

	go func() {
		<-c.ctx.Done()
		logger.Infof("srv stopped, stop etcd client together")
		cancel()
		_defaultCli.Close()
	}()

	return nil
}

func (e *Client) checkClient() error {
	if e.cli == nil {
		return fmt.Errorf("etcd client is not initialized yet")
	}

	return nil
}

func (e *Client) Close() {
	if e.cli != nil {
		e.cli.Close()
	}
}

func (e *Client) Find(searchedKey string) (resp *clientv3.GetResponse, err error) {
	if err = e.checkClient(); err != nil {
		return
	}

	opts := []clientv3.OpOption{clientv3.WithPrefix()}

	kv := clientv3.NewKV(e.cli)
	resp, err = kv.Get(e.ctx, searchedKey, opts...)
	if err != nil {
		return
	}

	return resp, err
}

func (e *Client) Get(key string, revision int64) (resp *clientv3.GetResponse, err error) {
	if err = e.checkClient(); err != nil {
		return
	}

	kv := clientv3.NewKV(e.cli)

	if revision != 0 {
		resp, err = kv.Get(e.ctx, key, clientv3.WithRev(revision))
	} else {
		resp, err = kv.Get(e.ctx, key)
	}

	if err != nil {
		switch err {
		case context.Canceled:
			logger.Errorf("ctx is canceled by another routine: %v", err)
		case context.DeadlineExceeded:
			logger.Errorf("ctx is attached with a deadline is exceeded: %v", err)
		case rpctypes.ErrEmptyKey:
			logger.Errorf("client-side error: %v", err)
		default:
			logger.Errorf("bad cluster endpoints, which are not etcd servers: %v", err)
		}

		return resp, err
	}

	if resp.Count == 0 {
		return resp, fmt.Errorf("%s not found", key)
	}
	return resp, err
}

func (e *Client) Put(key, value string) (resp *clientv3.PutResponse, err error) {
	if err = e.checkClient(); err != nil {
		return
	}

	kv := clientv3.NewKV(e.cli)
	updated, err := kv.Put(e.ctx, key, value)
	if err != nil {
		logger.Error(err)
		return
	}

	return updated, err
}

func (e *Client) Delete(key string) (err error) {
	if err = e.checkClient(); err != nil {
		return
	}

	kv := clientv3.NewKV(e.cli)
	_, err = kv.Delete(e.ctx, key)
	if err != nil {
		logger.Error(err)
		return
	}

	return err
}

func (e *Client) Lock(key, taskKey string) (grantResp *clientv3.LeaseGrantResponse, err error) {
	logger.Debugf("lock task %s with key %s", taskKey, key)
	if err = e.checkClient(); err != nil {
		return
	}

	kv := clientv3.NewKV(e.cli)
	lease := clientv3.NewLease(e.cli)
	granted, err := lease.Grant(e.ctx, 60)
	if err != nil {
		return
	}

	txn := kv.Txn(e.ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, "lock", clientv3.WithLease(granted.ID)), clientv3.OpDelete(taskKey)).
		Else()
	txnResp, err := txn.Commit()
	if err != nil {
		return
	}
	if !txnResp.Succeeded {
		return granted, fmt.Errorf("lock failed")
	}

	return granted, err
}
