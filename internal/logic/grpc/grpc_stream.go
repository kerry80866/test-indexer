package grpc

import (
	"context"
	"crypto/tls"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/svc"
	"errors"
	"fmt"
	"google.golang.org/grpc/metadata"
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
	xToken                string                        // 认证用的 x-token
	streamPingIntervalSec int                           // Stream心跳包发送间隔（秒）
	blockChan             chan *pb.SubscribeUpdateBlock // 区块数据通道
	connCtx               context.Context               // 当前连接的 context
	connCancel            context.CancelFunc            // 当前连接的 cancel 函数
	reconnectInterval     time.Duration                 // 每次重连之间的最小间隔（秒）
	sendTimeoutSec        int                           // gRPC Send 超时时间（秒）
	recvTimeoutSec        int                           // gRPC Recv 超时时间（秒）
	maxLatencyWarnMs      int                           // 区块延迟超 3 秒打 warning
	maxLatencyDropMs      int                           // 区块延迟超 5 秒断流重连
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
		sendTimeoutSec:        grpcConf.SendTimeoutSec,
		maxLatencyWarnMs:      grpcConf.MaxLatencyWarnMs,
		maxLatencyDropMs:      grpcConf.MaxLatencyDropMs,
	}, nil
}

func (m *GrpcStreamManager) Start() {
	m.mustConnect()
}

func (m *GrpcStreamManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopped = true // 标记已停止，必须在 cancel 之前设置，防止重入

	// 先 cancel context，通知所有 goroutine 退出（如 pingLoop, blockRecvLoop）
	if m.connCancel != nil {
		m.connCancel()
		m.connCancel = nil
	}

	// 再关闭 stream，确保没有 goroutine 在调用 Send()
	if m.stream != nil {
		if err := m.stream.CloseSend(); err != nil {
			logger.Warnf("[GrpcStream] CloseSend failed: %v", err)
		}
		m.stream = nil
	}

	// 最后关闭连接
	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			logger.Warnf("[GrpcStream] conn.Close failed: %v", err)
		}
		m.conn = nil
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
		logger.Infof("[GrpcStream] Connecting... Attempt %d", m.reconnectAttempts+1)
		m.reconnectAttempts++
		err := m.connect()
		if err == nil {
			return // 连接成功
		}
		logger.Errorf("[GrpcStream] Connect failed: %v, will retry...", err)
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
	// 再关闭 stream
	if m.stream != nil {
		if err := m.stream.CloseSend(); err != nil {
			logger.Warnf("[GrpcStream] CloseSend failed: %v", err)
		}
		m.stream = nil
	}

	// 开始建立新的stream
	m.connCtx, m.connCancel = context.WithCancel(context.Background())

	logger.Infof("[GrpcStream] Attempting to connect...")

	metaCtx := metadata.NewOutgoingContext(
		m.connCtx,
		metadata.New(map[string]string{"x-token": m.xToken}),
	)
	stream, err := m.client.Subscribe(metaCtx)
	if err != nil {
		logger.Errorf("[GrpcStream] Failed to subscribe: %v", err)
		return err // 只返回错误
	}

	req := buildSubscribeRequest()
	err = sendWithTimeout(m.connCtx, stream.Send, req, time.Duration(m.sendTimeoutSec)*time.Second)
	if err != nil {
		logger.Errorf("[GrpcStream] Failed to send request: %v", err)
		return err // 只返回错误
	}

	m.stream = stream
	m.reconnectAttempts = 0
	logger.Infof("[GrpcStream] Connection established")

	// 启动 ping 协程
	go m.pingLoop(m.connCtx)
	// 启动 block 监听协程
	go m.blockRecvLoop(m.connCtx)

	return nil
}

func (m *GrpcStreamManager) blockRecvLoop(ctx context.Context) {
	const warnSlotStep = 50
	var lastSlot uint64 = 0
	var lastWarnSlot uint64 = 0
	var totalLatency int64 = 0
	var count int64 = 0

	last := time.Now()
	warnThreshold := int64(m.maxLatencyWarnMs)
	dropThreshold := int64(m.maxLatencyDropMs)
	recvTimeout := time.Duration(m.recvTimeoutSec) * time.Second

	for {
		select {
		case <-ctx.Done():
			return // 优雅退出
		default:
			update, err := recvWithTimeout[*pb.SubscribeUpdate](ctx, m.stream.Recv, recvTimeout)
			now := time.Now()
			if err != nil {
				logger.Errorf("[GrpcStream] Stream error: %v", err)
				m.reconnect()
				return
			}

			switch u := (*update).GetUpdateOneof().(type) {
			case *pb.SubscribeUpdate_Block:
				// 检查是否丢失slot
				if lastSlot != 0 && u.Block.Slot != lastSlot+1 {
					logger.Errorf("[GrpcStream] slot skipped: last slot = %d, current slot = %d", lastSlot, u.Block.Slot)
				}
				lastSlot = u.Block.Slot

				blockTime := u.Block.BlockTime.Timestamp * 1000
				interval := now.UnixMilli() - blockTime // 算出收到这个区块时的延迟（ms）
				totalLatency += interval
				count++
				avgLatency := totalLatency / count
				logger.Infof("[GrpcStream] slot = %d, latency = %d ms, avg = %d ms (count = %d)", u.Block.Slot, interval, avgLatency, count)

				select {
				case m.blockChan <- u.Block:
					// 成功写入，无事发生
				default:
					//logger.Warnf("[GrpcStream] blockChan is full, discard block at slot %v", u.Block.Slot)
				}

				//无论是否写入成功，都要更新 last
				last = now
				if interval > dropThreshold {
					logger.Errorf("[GrpcStream] slot=%d, latency too high: %dms > %dms, reconnecting", u.Block.Slot, interval, dropThreshold)
					m.reconnect()
					return
				} else if interval > warnThreshold && lastSlot-lastWarnSlot >= warnSlotStep {
					logger.Warnf("[GrpcStream] slot=%d, high latency: %dms > %dms", u.Block.Slot, interval, warnThreshold)
					lastWarnSlot = lastSlot
				}
			}

			if time.Since(last) > recvTimeout {
				logger.Errorf("[GrpcStream] %v未收到block，触发重连", recvTimeout)
				m.reconnect()
				return
			}
		}
	}
}

// sendWithTimeout 向 sendFunc 发送带参数的请求，并在指定超时时间内等待返回。
// 使用 goroutine 执行 sendFunc，避免其阻塞主线程。
// 支持捕获 panic、自动取消、主 context 的提前终止。
func sendWithTimeout[T any](ctx context.Context, sendFunc func(T) error, req T, timeout time.Duration) error {
	// 创建子 context 以控制超时时间
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 创建结果通道，用于异步接收 sendFunc 的结果
	done := make(chan error, 1)

	// 在 goroutine 中执行 sendFunc，以避免其阻塞主线程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// 捕获 sendFunc 内部 panic 并写入 error 结果
				select {
				case done <- fmt.Errorf("sendFunc panic: %v", r):
				case <-timeoutCtx.Done():
					// 如果超时已发生，不写入，防止阻塞
				}
			}
		}()

		// 调用 sendFunc
		err := sendFunc(req)

		// 将返回结果写入 done（或放弃写入以避免阻塞）
		select {
		case done <- err:
		case <-timeoutCtx.Done():
		}
	}()

	// 等待 sendFunc 结果、主 context 被取消
	select {
	case err := <-done:
		return err // 正常返回或 panic 后的 error
	case <-ctx.Done():
		return ctx.Err() // 外部主动取消
	}
}

// result 是泛型结构体，用于包装 recvWithTimeout 的返回值和错误
type result[T any] struct {
	resp T     // 实际接收到的数据
	err  error // 可能的错误（包括 recvFunc 返回的 error 或 panic）
}

// recvWithTimeout 封装对 recvFunc 的调用，支持以下特性：
// - 指定超时时间（timeout）控制函数最大阻塞时间；
// - 可恢复 panic，防止函数内部崩溃影响主逻辑；
// - 响应主调用方 ctx 的取消信号（如服务 shutdown、手动取消）；
//
// 常用于 gRPC 等场景下的 Recv 操作，防止 Recv 永久阻塞。
func recvWithTimeout[T any](
	ctx context.Context, // 上层业务的控制 context（如服务级别 context）
	recvFunc func() (T, error), // 实际执行的阻塞性函数（如 stream.Recv()）
	timeout time.Duration, // 超时时间阈值
) (T, error) {
	// timeoutCtx 是基于 ctx 创建的子 context，用于实现本次调用的超时控制
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel() // 确保超时资源被释放

	// 创建带缓冲的通道，接收异步 goroutine 返回的结果
	done := make(chan result[T], 1)

	// 启动 goroutine 来调用 recvFunc，避免其阻塞主线程
	go func() {
		// 使用 recover 捕获 recvFunc 内部的 panic
		defer func() {
			if r := recover(); r != nil {
				var zero T // 零值用于 panic 时返回
				select {
				case done <- result[T]{zero, fmt.Errorf("recvFunc panic: %v", r)}:
				case <-timeoutCtx.Done():
					// 如果已经超时，放弃写入，避免阻塞
				}
			}
		}()

		// 实际调用 recvFunc（如 stream.Recv()）
		resp, err := recvFunc()

		// 将结果写入通道；如果已超时则丢弃
		select {
		case done <- result[T]{resp, err}:
		case <-timeoutCtx.Done():
			// 超时后不写入，避免死锁
		}
	}()

	// 主逻辑等待三种退出信号之一：
	select {
	case result := <-done:
		// 正常接收到结果或 panic 错误
		return result.resp, result.err

	case <-timeoutCtx.Done():
		// 本次调用超时（注意：ctx.Done() 也会触发）
		var zero T
		return zero, timeoutCtx.Err()

	case <-ctx.Done():
		// 主业务主动取消（如 stop/reconnect）
		var zero T
		return zero, ctx.Err()
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
				logger.Warnf("[GrpcStream] Ping failed (non-critical): %v", err)
				continue // ✅ 安全！由 recvLoop 兜底 reconnect
			}
		}
	}
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
