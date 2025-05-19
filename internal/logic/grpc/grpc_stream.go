package grpc

import (
	"context"
	"crypto/tls"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/svc"
	"errors"
	"fmt"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type GrpcStreamManager struct {
	mu                    sync.Mutex                    // 互斥锁，保护并发安全
	conn                  *grpc.ClientConn              // gRPC 连接对象
	client                pb.GeyserClient               // gRPC 客户端
	stream                pb.Geyser_SubscribeClient     // gRPC 订阅流
	stopped               bool                          // 标记是否已经停止
	reconnectAttempts     int                           // 已重连次数
	reconnectInterval     time.Duration                 // 重连基础间隔
	xToken                string                        // 认证用的 x-token
	streamPingIntervalSec int                           // Stream心跳包发送间隔（秒）
	blockChan             chan *pb.SubscribeUpdateBlock // 区块数据通道
	connCtx               context.Context               // 当前连接的 context
	connCancel            context.CancelFunc            // 当前连接的 cancel 函数
	recvTimeoutSec        int                           // Recv 超时时间（秒）
	blockRecvTimeoutSec   int                           // block接收超时时间（秒）
	sendTimeoutSec        int                           // gRPC发送超时时间（秒）
}

func NewGrpcStreamManager(sc *svc.GrpcServiceContext, blockChan chan *pb.SubscribeUpdateBlock) (*GrpcStreamManager, error) {
	grpcConf := sc.Config.Grpc

	configTls := &tls.Config{
		InsecureSkipVerify: true,
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), time.Duration(grpcConf.ConnectTimeoutSec)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		grpcConf.Endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(configTls)),
		grpc.WithInitialWindowSize(int32(grpcConf.InitialWindowSize)),
		grpc.WithInitialConnWindowSize(int32(grpcConf.InitialConnWindowSize)),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(grpcConf.MaxCallSendMsgSize),
			grpc.MaxCallRecvMsgSize(grpcConf.MaxCallRecvMsgSize),
		),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(grpcConf.KeepalivePingIntervalSec) * time.Second,
			Timeout:             time.Duration(grpcConf.KeepalivePingTimeoutSec) * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return &GrpcStreamManager{
		conn:                  conn,
		client:                pb.NewGeyserClient(conn),
		reconnectAttempts:     0,
		reconnectInterval:     time.Duration(grpcConf.ReconnectIntervalSec) * time.Second,
		xToken:                grpcConf.XToken,
		streamPingIntervalSec: grpcConf.StreamPingIntervalSec,
		blockChan:             blockChan,
		recvTimeoutSec:        grpcConf.RecvTimeoutSec,
		blockRecvTimeoutSec:   grpcConf.BlockRecvTimeoutSec,
		sendTimeoutSec:        grpcConf.SendTimeoutSec,
	}, nil
}

func (m *GrpcStreamManager) Start() {
	m.mustConnect()
}

func (m *GrpcStreamManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopped = true // 标记已停止
	if m.connCancel != nil {
		m.connCancel() // 🔥 建议加上
		m.connCancel = nil
	}
	if m.conn != nil {
		err := m.conn.Close()
		_ = err
	}
}

// 内部循环直到连接成功
func (m *GrpcStreamManager) mustConnect() {
	for {
		m.mu.Lock()
		if m.stopped {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		if m.reconnectAttempts > 0 {
			if m.reconnectAttempts > 3 {
				time.Sleep(m.reconnectInterval * 2)
			} else {
				time.Sleep(m.reconnectInterval)
			}
		}
		log.Printf("Connecting... Attempt %d", m.reconnectAttempts+1)
		m.reconnectAttempts++
		err := m.connect()
		if err == nil {
			return // 连接成功
		}
		log.Printf("Connect failed: %v, will retry...", err)
	}
}

func buildSubscribeRequest() *pb.SubscribeRequest {
	blocks := make(map[string]*pb.SubscribeRequestFilterBlocks)
	blocks["blocks"] = &pb.SubscribeRequestFilterBlocks{
		AccountInclude:      consts.GrpcAccountInclude,
		IncludeTransactions: boolPtr(true),  // ✅ 保留转 SOL、swap、transfer 等交易
		IncludeAccounts:     boolPtr(false), // 不再收账户余额变化的单独 AccountUpdate（vote 省了）
		IncludeEntries:      boolPtr(false), // IncludeEntries 是 Solana 底层的日志，普通业务基本没用。
	}
	commitment := pb.CommitmentLevel_CONFIRMED
	return &pb.SubscribeRequest{
		Blocks:     blocks,
		Commitment: &commitment,
	}
}

// connect 只尝试一次连接
func (m *GrpcStreamManager) connect() error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return errors.New("manager is stopped")
	}
	defer m.mu.Unlock()

	// 先关闭旧的 context，优雅退出旧 goroutine
	if m.connCancel != nil {
		m.connCancel()
		m.connCancel = nil
	}
	m.connCtx, m.connCancel = context.WithCancel(context.Background())

	log.Println("Attempting to connect...")

	metaCtx := metadata.NewOutgoingContext(
		m.connCtx,
		metadata.New(map[string]string{"x-token": m.xToken}),
	)
	stream, err := m.client.Subscribe(metaCtx)
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
		return err // 只返回错误
	}

	req := buildSubscribeRequest()
	err = sendWithTimeout(m.connCtx, stream.Send, req, time.Duration(m.sendTimeoutSec)*time.Second)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return err // 只返回错误
	}

	m.stream = stream
	m.reconnectAttempts = 0
	log.Println("Connection established")

	// 启动 ping 协程
	go m.pingLoop(m.connCtx)
	// 启动 block 监听协程
	go m.blockRecvLoop(m.connCtx)

	return nil
}

func (m *GrpcStreamManager) blockRecvLoop(ctx context.Context) {
	last := time.Now()
	blockTimeout := time.Duration(m.blockRecvTimeoutSec) * time.Second
	for {
		select {
		case <-ctx.Done():
			return // 优雅退出
		default:
			update, err := m.stream.Recv() //recvWithTimeout[*pb.SubscribeUpdate](ctx, m.stream.Recv, time.Duration(m.recvTimeoutSec)*time.Second)
			now := time.Now()
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Printf("Stream closed by server (EOF), will reconnect")
					m.reconnect()
					return
				}

				log.Printf("Stream error: %v", err)
				if m.reconnectIfBlockTimeout(last, blockTimeout) {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			switch u := (*update).GetUpdateOneof().(type) {
			case *pb.SubscribeUpdate_Block:
				interval := now.UnixMilli() - u.Block.BlockTime.Timestamp*1000 // 算出你收到这个区块时的延迟（ms）
				log.Printf("received block at slot %v, latency to blockTime: %v ms", u.Block.Slot, interval)

				//select {
				//case m.blockChan <- u.Block:
				//	// 成功写入，无事发生
				//default:
				//	log.Printf("blockChan is full, discard block at slot %v", u.Block.Slot)
				//}
				//interval1 := now.Sub(last)
				//log.Printf("received block at slot %v, interval since last block: %v ms", u.Block.Slot, interval1.Milliseconds())
				// 无论是否写入成功，都要更新 last
				last = now
			}
		}

		if m.reconnectIfBlockTimeout(last, blockTimeout) {
			return
		}
	}
}

// 带超时的 Send
func sendWithTimeout[T any](ctx context.Context, sendFunc func(T) error, req T, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- sendFunc(req)
	}()

	select {
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	case err := <-done:
		return err
	}
}

// 带超时的 Recv
func recvWithTimeout[T any](ctx context.Context, recvFunc func() (T, error), timeout time.Duration) (T, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan struct {
		resp T
		err  error
	}, 1)

	go func() {
		resp, err := recvFunc()
		done <- struct {
			resp T
			err  error
		}{resp, err}
	}()

	select {
	case <-timeoutCtx.Done():
		var zero T
		return zero, timeoutCtx.Err()
	case result := <-done:
		return result.resp, result.err
	}
}

// 心跳检测
func (m *GrpcStreamManager) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.streamPingIntervalSec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return // 优雅退出
		case <-ticker.C:
			pingReq := &pb.SubscribeRequest{
				Ping: &pb.SubscribeRequestPing{Id: 1},
			}
			err := sendWithTimeout(ctx, m.stream.Send, pingReq, time.Duration(m.sendTimeoutSec)*time.Second)
			if err != nil {
				log.Printf("Ping failed: %v", err)
				// 这里只记录日志，不触发重连
			}
		}
	}
}

func (m *GrpcStreamManager) reconnectIfBlockTimeout(last time.Time, timeout time.Duration) bool {
	if time.Since(last) > timeout {
		log.Printf("%v未收到block，触发重连", timeout)
		m.reconnect()
		return true
	}
	return false
}

func (m *GrpcStreamManager) reconnect() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	if m.connCancel != nil {
		m.connCancel() // 关闭所有相关 goroutine
		m.connCancel = nil
	}
	m.mu.Unlock()

	go m.mustConnect()
}

func boolPtr(b bool) *bool {
	return &b
}

// main 函数已注释，改为提供 New、Start、Stop 接口
/*
func main() {
	ctx := context.Background()

	// Create manager with data handler and x-token
	manager, err := NewGrpcStreamManager(
		"your-grpc-url:2053",
		"your-x-token",
		handleAccountUpdate,
		10,
		1<<30,
		1<<30,
		64*1024*1024,
		64*1024*1024,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create subscription request for blocks and slots
	blocks := make(map[string]*pb.SubscribeRequestFilterBlocks)
	blocks["blocks"] = &pb.SubscribeRequestFilterBlocks{
		AccountInclude:      []string{},
		IncludeTransactions: boolPtr(true),
		IncludeAccounts:     boolPtr(true),
		IncludeEntries:      boolPtr(false),
	}

	slots := make(map[string]*pb.SubscribeRequestFilterSlots)
	slots["slots"] = &pb.SubscribeRequestFilterSlots{
		FilterByCommitment: boolPtr(true),
	}

	commitment := pb.CommitmentLevel_CONFIRMED
	req := &pb.SubscribeRequest{
		Blocks:     blocks,
		Slots:      slots,
		Commitment: &commitment,
	}

	log.Println("Starting block and slot monitoring...")
	log.Println("Monitoring blocks for Token Program, System Program, and Wrapped SOL activities...")

	// Connect and handle updates
	if err := manager.Connect(ctx, req); err != nil {
		log.Fatal(err)
	}

	// Keep the main goroutine running
	select {}
}
*/
