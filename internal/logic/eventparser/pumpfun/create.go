package pumpfun

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/logic/eventparser/common"
	"github.com/dex-indexer-sol/internal/tools"
	"github.com/dex-indexer-sol/pb"
	"github.com/dex-indexer-sol/pkg/logger"
	"github.com/dex-indexer-sol/pkg/types"
	"github.com/near/borsh-go"
	"runtime/debug"
)

type PumpCreateEvent struct {
	Sign                 uint64
	Name                 string
	Symbol               string
	Uri                  string
	Mint                 types.Pubkey
	BondingCurve         types.Pubkey
	User                 types.Pubkey
	Creator              types.Pubkey
	Timestamp            uint64
	VirtualTokenReserves uint64
	VirtualSolReserves   uint64
	RealTokenReserves    uint64
	TokenTotalSupply     uint64
}

// 示例交易：https://solscan.io/tx/5hHxQWz2H7FNv7oNoDHTmqhAGZLDjW5dTgX1sazUAPnXoY5MqirHnV5oPoJPQSAH6A11ynzKeqmTVB7Cy52iTbcK
//
// Pump.fun - Create 指令账户布局：
//
// #0  - Mint 账户（新创建的 Token Mint，用于市场）
// #1  - Mint Authority（Mint 的权限地址，由程序派生）
// #2  - Bonding Curve 主账户（池子地址）
// #3  - Bonding Curve Vault（池子 TokenAccount）
// #4  - Global 配置账户（全局状态管理）
// #5  - Metaplex Token Metadata 程序地址
// #6  - Metadata 账户（Metaplex 生成的元数据 PDA）
// #7  - 用户钱包地址
// #8  - System Program（系统程序）
// #9  - Token Program（SPL Token 程序）
// #10 - Associated Token Program（创建 ATA 用）
// #11 - Rent 账户（租金信息）
// #12 - Event Authority（事件权限 PDA，用于记录事件）
// #13 - Pump.fun 程序地址（Program ID）
func extractCreateEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (next int) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Pumpfun:Create] panic: %v, stack=%s, tx=%s", r, debug.Stack(), ctx.TxHashString())
			next = -1
		}
	}()

	ix := instrs[current]

	// 1. 校验指令结构
	if len(ix.Accounts) < 14 {
		logger.Errorf("[Pumpfun:Create] 指令账户长度不足: got=%d, expect>=14, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}
	if len(ix.Data) < 24 {
		logger.Errorf("[Pumpfun:Create] 指令数据过短: got=%d, expect>=24, tx=%s",
			len(ix.Data), ctx.TxHashString())
		return -1
	}

	// 2. 提取并解析事件
	eventIndex := findEventInstruction(instrs, current, ix.Accounts[12]) // Event Authority
	if eventIndex < 0 {
		// 加上预期 EventAuthority
		logger.Errorf("[Pumpfun:Create] 未找到事件日志指令: authority=%s, tx=%s", ix.Accounts[12], ctx.TxHashString())
		return -1
	}
	eventIx := instrs[eventIndex]
	event := PumpCreateEvent{}
	if err := borsh.Deserialize(&event, eventIx.Data[8:]); err != nil {
		logger.Errorf("[Pumpfun:Create] 事件反序列化失败: %v, tx=%s", err, ctx.TxHashString())
		return -1
	}

	// 3. 校验交易 token mint
	if event.Mint != ix.Accounts[0] {
		logger.Errorf("[Pumpfun:Create] mint 不匹配 (expected=%s, got=%s): tx=%s", ix.Accounts[0], event.Mint, ctx.TxHashString())
		return -1
	}

	// 4. 校验BondingCurve地址一致性
	if event.BondingCurve != ix.Accounts[2] {
		logger.Errorf("[Pumpfun:Create] BondingCurve 不一致 (expected=%s, got=%s): tx=%s", event.BondingCurve, ix.Accounts[2], ctx.TxHashString())
		return -1
	}

	// 5. 校验用户地址一致性
	if event.User != ix.Accounts[7] {
		logger.Errorf("[Pumpfun:Create] 用户地址不匹配: expected=%s, got=%s, tx=%s", ix.Accounts[7], event.User, ctx.TxHashString())
		return -1
	}

	// 6. 校验 Token Program 是否为 SPL Token
	if !tools.IsSPLTokenProgram(ix.Accounts[9]) {
		logger.Errorf("[Pumpfun:Create] Token Program 非 SPL 标准程序: got=%s, tx=%s", ix.Accounts[9], ctx.TxHashString())
		return -1
	}

	// 7. 提取池子主账户与其持有的 token 账户
	poolAddress := ix.Accounts[2]      // Bonding Curve 主账户
	poolTokenAccount := ix.Accounts[3] // Bonding Curve Vault（池子 token account）

	// 获取池子的 SOL 余额（Post 状态）
	poolSolBalance, ok := ctx.Tx.SolBalances[poolAddress]
	if !ok {
		logger.Errorf("[Pumpfun:Create] 缺失池子 SOL 余额: account=%s, tx=%s", poolAddress, ctx.TxHashString())
		return -1
	}

	// 获取池子的 token 余额（Post 状态）
	poolTokenBalance, ok := ctx.Balances[poolTokenAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Create] 缺失池子 token 余额: account=%s, tx=%s", poolTokenAccount, ctx.TxHashString())
		return -1
	}

	// 8. 校验池子持有的 token account 所有者是否为池子主账户
	if poolTokenBalance.PostOwner != poolAddress {
		logger.Errorf("[Pumpfun:Create] pool token account 所有者异常: expected=%s, actual=%s, account=%s, tx=%s",
			poolAddress, poolTokenBalance.PostOwner, poolTokenAccount, ctx.TxHashString())
		return -1
	}

	// 9. 获取创建者的 SOL 余额
	var creatorSolBalance uint64
	if bal, ok := ctx.Balances[event.Creator]; ok {
		creatorSolBalance = bal.PostBalance
	}

	// 10. 构建 CreatePool 类型的 TradeEvent
	poolEvent := &pb.LiquidityEvent{
		Type:      pb.EventType_CREATE_POOL,
		EventId:   core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:      ctx.Slot,
		BlockTime: ctx.BlockTime,
		TxHash:    ctx.TxHash,
		Signers:   ctx.Signers,
		Dex:       consts.DexPumpfun,

		UserWallet:  event.User[:],
		PairAddress: poolAddress[:],

		TokenDecimals: uint32(poolTokenBalance.Decimals),
		QuoteDecimals: 9, // SOL 精度固定为 9

		TokenAmount:      0,
		QuoteTokenAmount: 0,

		Token:      event.Mint[:],
		QuoteToken: consts.NativeSOLMint[:], // quote token（SOL）

		TokenAccount:      poolTokenAccount[:],
		QuoteTokenAccount: poolAddress[:], // 对于 SOL，没有 token account，直接使用主账户地址。

		TokenAccountOwner:      poolTokenBalance.PostOwner[:],
		QuoteTokenAccountOwner: poolAddress[:], // 对于 SOL 主账户，其 owner 是自己

		PairTokenBalance: poolTokenBalance.PostBalance,
		PairQuoteBalance: poolSolBalance.PostBalance,

		UserTokenBalance: 0,
		UserQuoteBalance: creatorSolBalance,
	}

	// 11. 将池子的 SOL 余额补充进 token balance 列表，便于后续统一处理
	ctx.Tx.AppendSolToTokenBalances(poolSolBalance)

	// 12. 构造CreatePool标准事件结构
	createPool := &core.Event{
		ID:        poolEvent.EventId,
		EventType: uint32(poolEvent.Type),
		Key:       poolEvent.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{Liquidity: poolEvent},
		},
	}

	// 13. 衍生出 LaunchpadTokenEvent 事件
	tokenEvent := &pb.LaunchpadTokenEvent{
		Type:      pb.EventType_LAUNCHPAD_TOKEN,
		EventId:   createPool.ID + 1,
		Slot:      ctx.Slot,
		BlockTime: ctx.BlockTime,
		TxHash:    ctx.TxHash,
		Signers:   ctx.Signers,
		Dex:       consts.DexPumpfun,

		UserWallet: event.User[:],
		Creator:    event.Creator[:],

		Decimals: uint32(poolTokenBalance.Decimals),

		TotalSupply: event.TokenTotalSupply,
		Token:       event.Mint[:],
		PairAddress: poolAddress[:],

		Symbol: event.Symbol,
		Name:   event.Name,
		Uri:    event.Uri,
	}
	launchpadTokenEvent := &core.Event{
		ID:        tokenEvent.EventId,
		EventType: uint32(tokenEvent.Type),
		Key:       tokenEvent.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Token{
				Token: tokenEvent,
			},
		},
	}

	// 14. 衍生出 AddLiquidity 事件（CreatePool 隐含的首笔注入逻辑，不对应链上指令））
	liquidityEvent := common.CloneLiquidityEvent(createPool)
	liquidityEvent.ID += 2
	liquidityEvent.Event.GetLiquidity().EventId = liquidityEvent.ID
	liquidityEvent.EventType = uint32(pb.EventType_ADD_LIQUIDITY)
	liquidityEvent.Event.GetLiquidity().Type = pb.EventType_ADD_LIQUIDITY
	liquidityEvent.Event.GetLiquidity().TokenAmount = event.TokenTotalSupply
	liquidityEvent.Event.GetLiquidity().QuoteTokenAmount = 0 // 首次建池并未注入 SOL，SOL 注入发生在后续 Buy

	ctx.AddEvent(createPool)
	ctx.AddEvent(launchpadTokenEvent)
	ctx.AddEvent(liquidityEvent)
	return eventIndex + 1
}
