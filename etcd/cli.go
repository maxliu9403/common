package etcd

import (
	"context"
	"fmt"
	"github.com/maxliu9403/common/logger"
	"github.com/samber/lo"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
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
	ctx             context.Context
	cli             *clientv3.Client
	leaseCancelFunc map[clientv3.LeaseID]context.CancelFunc // 控制解锁后释放续约
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
	logger.Info("connecting to etcd... ")
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

	// 健康检测
	//ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//defer cancel()
	//
	//_, err := e.cli.Get(ctx, "non-existent-key-for-health-check")
	//if err != nil {
	//	return fmt.Errorf("failed to communicate with etcd: %v", err)
	//}

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

// Get
// revision 指定版本
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

/*

在分布式系统中，续约是一个重要的机制，尤其是当我们使用基于租约的锁或其他资源时。以下是为什么需要续约的几个原因：

1.处理长时间运行的任务：当一个客户端获取锁并开始执行任务时，如果任务的执行时间超过了锁的租约时间，那么锁会在任务完成之前自动过期。
这可能会导致其他客户端在第一个任务完成之前获取锁，从而引发数据不一致或其他问题。通过定期续约，我们可以确保锁在任务完成之前不会过期。

2.容错和网络不稳定：在分布式系统中，网络延迟和短暂的网络中断是常见的。如果一个客户端因为网络问题暂时与etcd失去了连接，
但在租约时间内重新连接，那么它应该有机会续约其锁，而不是立即失去它。

3.避免"僵尸"锁：在某些情况下，持有锁的客户端可能会崩溃或被意外终止。
如果没有租约机制，这个锁可能会永远存在，导致资源被永久锁定。通过设置租约，我们可以确保这种"僵尸"锁在一段时间后自动释放，从而允许其他客户端获取资源。

4.提供更多的灵活性：续约机制允许客户端基于其当前的工作负载和资源需求动态地调整锁的持有时间。
例如，如果一个客户端知道它需要更长的时间来完成任务，它可以选择续约锁，而不是释放它并重新获取。

5.减少资源争用：在高并发的环境中，多个客户端可能会频繁地尝试获取和释放锁。
通过允许客户端续约其锁，我们可以减少锁的争用，从而提高系统的整体效率。

总的来说，续约机制为分布式锁提供了一个有效的方式来管理资源的生命周期，确保数据的一致性，并提高系统的可靠性和效率。

*/
func (e *Client) keepAlive(leaseCtx context.Context, ttl int64, leaseID clientv3.LeaseID) {
	// 特殊case，避免interval=0时触发NewTicker的Panic
	interval := ttl / 2
	if interval <= 0 {
		interval = 1
	}

	// 1/2 ttl时间时开始续租
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		// 监听续租被释放
		case <-leaseCtx.Done():
			return
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			if _, err := e.cli.KeepAliveOnce(e.ctx, leaseID); err != nil {
				logger.Errorf("failed to renew lease: %v, leaseID:%v", err.Error(), leaseID)
				return
			}
		}
	}
}

// TryLockBlocking 阻塞等待
// ttl:加锁时间（秒）0表示永不过期
// timeout 等待加锁时间
func (e *Client) TryLockBlocking(key string, ttl int64, timeout time.Duration) (grantResp *clientv3.LeaseGrantResponse, err error) {
	timeoutCtx, cancel := context.WithTimeout(e.ctx, timeout)
	defer cancel()

	if err = e.checkClient(); err != nil {
		return
	}

	e.leaseCancelFunc = map[clientv3.LeaseID]context.CancelFunc{}
	shouldContinueWatching := true // 用于控制外部循环
	for {
		select {
		case <-e.ctx.Done():
			return nil, fmt.Errorf("failed to acquire etcd lock, ctx done %v: %v", timeout, e.ctx.Err())
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("failed to acquire etcd lock within %v: %v", timeout, timeoutCtx.Err())
		default:
			// 创建租约
			lease, err := e.cli.Grant(e.ctx, ttl)
			if err != nil {
				return nil, fmt.Errorf("failed to grant lease: %v", err)
			}

			txn := clientv3.NewKV(e.cli).Txn(e.ctx)
			// 判断key是否存在，不存在将 key 设置为当前时间，并关联到前面创建的租约
			txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
				Then(clientv3.OpPut(key, time.Now().String(), clientv3.WithLease(lease.ID))).
				Else()
			txnResp, err := txn.Commit()
			if err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %v", err)
			}

			if txnResp.Succeeded {
				// 为每个续约创建一个ctx，便于unlock时取消续约
				leaseCtx, leaseCancel := context.WithCancel(e.ctx)
				e.leaseCancelFunc[lease.ID] = leaseCancel
				// 续约
				go e.keepAlive(leaseCtx, ttl, lease.ID)
				return lease, nil
			}

			// 如果锁被占用，使用watch监听锁释放
			watchCh := e.cli.Watch(timeoutCtx, key)
			for item := range watchCh {
				lo.ForEach(item.Events, func(ev *clientv3.Event, _ int) {
					shouldContinueWatching = lo.If(ev.Type == clientv3.EventTypeDelete, false).Else(true)
				})
				if !shouldContinueWatching {
					break
				}
			}

		}
	}
}

// TryLock 非阻塞
// ttl:加锁时间（秒）0表示永不过期
func (e *Client) TryLock(key string, ttl int64) (grantResp *clientv3.LeaseGrantResponse, err error) {
	if err = e.checkClient(); err != nil {
		return
	}

	e.leaseCancelFunc = map[clientv3.LeaseID]context.CancelFunc{}

	// 创建租约
	lease, err := e.cli.Grant(e.ctx, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to grant lease: %v", err)
	}

	txn := clientv3.NewKV(e.cli).Txn(e.ctx)
	// 判断key是否存在，不存在将 key 设置为当前时间，并关联到前面创建的租约
	txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, time.Now().String(), clientv3.WithLease(lease.ID))).
		Else()
	txnResp, err := txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	if txnResp.Succeeded {
		// 为每个续约创建一个ctx，便于unlock时取消续约
		leaseCtx, leaseCancel := context.WithCancel(e.ctx)
		e.leaseCancelFunc[lease.ID] = leaseCancel
		// 续约
		go e.keepAlive(leaseCtx, ttl, lease.ID)
		return lease, nil
	}

	// 如果锁被占用，立即返回一个错误
	return nil, fmt.Errorf("etcd lock is currently held by another client")
}

// Unlock 解锁续约
func (e *Client) Unlock(leaseID clientv3.LeaseID) error {
	// 释放续约
	if cancelFunc, exists := e.leaseCancelFunc[leaseID]; exists {
		cancelFunc()
		delete(e.leaseCancelFunc, leaseID)
	}
	if err := e.checkClient(); err != nil {
		return err
	}
	// 释放基于租约的锁，锁的生命周期与其关联的租约紧密相关
	if _, err := e.cli.Revoke(e.ctx, leaseID); err != nil {
		return fmt.Errorf("failed to revoke lease: %v", err)
	}
	return nil
}
