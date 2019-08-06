
package main

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	//"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

func main() {
	var (
		client       *clientv3.Client
		//err          error
		kv           clientv3.KV
		keepResp     *clientv3.LeaseKeepAliveResponse
		keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
	)

	client, _ = clientv3.New(clientv3.Config{
		Endpoints:   []string{"192.168.99.101:2379", "192.168.99.103:2379", "192.168.99.104:2379"},
		DialTimeout: 5 * time.Second,
	})
	//创建租约
	lease := clientv3.NewLease(client)
	//判断是否有问题
	if leaseRes, err := lease.Grant(context.TODO(), 20); err != nil {
		fmt.Println(err)
		return
	} else {
		//得到租约id
		leaseId := leaseRes.ID

		//定义一个上下文使得租约5秒过期
		ctx, _ := context.WithTimeout(context.TODO(), 5*time.Second)

		//自动续租（底层会每次讲租约信息扔到 <-chan *clientv3.LeaseKeepAliveResponse 这个管道中）
		if keepRespChan, err = lease.KeepAlive(ctx, leaseId); err != nil {
			fmt.Println(err)
			return
		}
		//启动一个新的协程来select这个管道
		go func() {
			for {
				select {
				case keepResp = <-keepRespChan:
					if keepResp == nil {
						fmt.Println("租约失效了")
						goto END //失效跳出循环
					} else {
						//每秒收到一次应答
						fmt.Println("收到租约应答", keepResp.ID)
					}

				}
			}
		END:
		}()
		//得到操作键值对的kv
		kv = clientv3.NewKV(client)
		//进行写操作
		if putResp, err := kv.Put(context.TODO(), "/cron/lock/job1", "123", clientv3.WithLease(leaseId) /*高速etcd这个key对应的租约*/); err != nil {
			fmt.Println(err)
			return
		} else {
			fmt.Println("写入成功", putResp.Header.Revision /*这东西你可以理解为每次操作的id*/)
		}
	}
	//监听这个key的租约是否过期
	for {
		getResp, err := kv.Get(context.TODO(), "/cron/lock/job1")
		if  err != nil {
			fmt.Println(err)
			return
		}

		if getResp.Count == 0 {
			fmt.Println("kv过期了")
			break
		}

		fmt.Println("kv没过期", getResp.Kvs)
		time.Sleep(2 * time.Second)

	}
}