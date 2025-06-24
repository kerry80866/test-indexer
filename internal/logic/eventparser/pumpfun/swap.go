package pumpfun

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/pb"
	"encoding/binary"
	"github.com/near/borsh-go"
	"runtime/debug"
)

type PumpSwapEvent struct {
	Sign                 uint64
	Mint                 types.Pubkey
	SolAmount            uint64
	TokenAmount          uint64
	IsBuy                bool
	User                 types.Pubkey
	Timestamp            uint64
	VirtualSolReserves   uint64
	VirtualTokenReserves uint64
	CurrentSolReserves   uint64
	CurrentTokenReserves uint64
}

// extractSwapEvent 解析 Pump.fun 的 swap 事件，构造标准 TradeEvent（BUY / SELL）。
// 示例交易：
// Sell: https://solscan.io/tx/3NCxJ1jNF1SHjjGKDxMhnzyqwSdEDoitPLzvEdfBZrTPXhxA21YkydApvP8rLzeM36Bpa2jWqnrgryhw9oqgBLpv
// Buy: https://solscan.io/tx/26N7CkAScr2msSTHNoEGtfwWkHwrsqRhwUPjh366SyYG5oY4CojjDQFZR8ZPN7nt5JEqqYBBvWndHxNQcf1mkBzz
//
// Pump.fun 交易账户结构：
//  0. Global 配置账户（不可变）
//  1. 手续费账户
//  2. 被购买代币的 Mint
//  3. Bonding Curve 主账户（池子地址）
//  4. Bonding Curve Vault（池子 TokenAccount）
//  5. 用户 Associated Token Account（User TokenAccount）
//  6. 用户主账户（用户地址）
//  7. System Program
//  8. Token Program
//  9. Creator Vault
//  10. Event Authority (事件地址)
//  11. Pump.fun 程序账户
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	isBuy bool,
) (next int) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Pumpfun:Swap] panic: %v, stack=%s, tx=%s", r, debug.Stack(), ctx.TxHashString())
			next = -1
		}
	}()

	ix := instrs[current]

	// 1. 校验指令结构
	if len(ix.Accounts) < 12 {
		logger.Errorf("[Pumpfun:Swap] 指令账户长度不足: got=%d, expect>=12, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}
	if len(ix.Data) < 24 {
		logger.Errorf("[Pumpfun:Swap] 指令数据过短: got=%d, expect>=24, tx=%s",
			len(ix.Data), ctx.TxHashString())
		return -1
	}

	// 2. 提取并解析事件
	eventIndex := findEventInstruction(instrs, current, ix.Accounts[10]) // Event Authority
	if eventIndex < 0 {
		// 加上预期 EventAuthority
		logger.Errorf("[Pumpfun:Swap] 未找到事件日志指令: tx=%s", ctx.TxHashString())
		return -1
	}
	eventIx := instrs[eventIndex]
	event := PumpSwapEvent{}
	if err := borsh.Deserialize(&event, eventIx.Data[8:]); err != nil {
		logger.Errorf("[Pumpfun:Swap] 事件反序列化失败: %v, tx=%s", err, ctx.TxHashString())
		return -1
	}

	// 3. 校验方向
	if event.IsBuy != isBuy {
		logger.Errorf("[Pumpfun:Swap] 事件方向不匹配 (expected %v, got %v): tx=%s", isBuy, event.IsBuy, ctx.TxHashString())
		return -1
	}

	// 4. 校验交易 token mint
	if event.Mint != ix.Accounts[2] {
		logger.Errorf("[Pumpfun:Swap] mint 不匹配 (expected=%s, got=%s): tx=%s", ix.Accounts[2], event.Mint, ctx.TxHashString())
		return -1
	}

	// 5. 校验用户地址一致性
	userWallet := ix.Accounts[6]
	if event.User != userWallet {
		logger.Errorf("[Pumpfun:Swap] 事件中用户地址不匹配: expected=%s, got=%s, tx=%s", userWallet, event.User, ctx.TxHashString())
		return -1
	}

	// 6. 校验 token 与 SOL 金额
	tokenAmount := binary.LittleEndian.Uint64(ix.Data[8:16])
	if event.TokenAmount != tokenAmount {
		logger.Errorf("[Pumpfun:Swap] Token 金额不匹配: event.TokenAmount=%d, expected=%d, tx=%s",
			event.TokenAmount, tokenAmount, ctx.TxHashString())
		return -1
	}

	var eventType pb.EventType
	if isBuy {
		maxSolAmount := binary.LittleEndian.Uint64(ix.Data[16:24])
		if event.SolAmount > maxSolAmount {
			logger.Errorf("[Pumpfun:Swap] SOL 金额超出最大值: event.SolAmount=%d, maxSolAmount=%d, tx=%s",
				event.SolAmount, maxSolAmount, ctx.TxHashString())
			return -1
		}
		eventType = pb.EventType_TRADE_BUY
	} else {
		minSolAmount := binary.LittleEndian.Uint64(ix.Data[16:24])
		if event.SolAmount < minSolAmount {
			logger.Errorf("[Pumpfun:Swap] SOL 金额低于最小值: event.SolAmount=%d, minSolAmount=%d, tx=%s",
				event.SolAmount, minSolAmount, ctx.TxHashString())
			return -1
		}
		eventType = pb.EventType_TRADE_SELL
	}

	// 7. 提取关键账户
	pairAddress := ix.Accounts[3]
	pairTokenAccount := ix.Accounts[4]
	userTokenAccount := ix.Accounts[5]

	pairSolBalance, ok := ctx.Tx.SolBalances[pairAddress]
	if !ok {
		logger.Errorf("[Pumpfun:Swap] 缺失 pair SOL 余额: account=%s (pairAddress), tx=%s", pairAddress, ctx.TxHashString())
		return -1
	}

	pairTokenBalance, ok := ctx.Balances[pairTokenAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Swap] 缺失 pair token 余额: account=%s (pairTokenAccount), tx=%s", pairTokenAccount, ctx.TxHashString())
		return -1
	}

	// 8. 校验 pair token account 的所有者是否为池子主账户（pairAddress）。
	// 	  正常情况下，池子持有的 token account 应由池子地址持有；若不一致，说明结构异常或数据错乱。
	if pairTokenBalance.PostOwner != pairAddress {
		logger.Errorf("[Pumpfun:Swap] pair token account 所有者异常: expected=%s, actual=%s, account=%s, tx=%s",
			pairAddress, pairTokenBalance.PostOwner, pairTokenAccount, ctx.TxHashString())
		return -1
	}

	userSolBalance, ok := ctx.Tx.SolBalances[userWallet]
	if !ok {
		logger.Errorf("[Pumpfun:Swap] 缺失 user SOL 余额: account=%s (userWallet), tx=%s", userWallet, ctx.TxHashString())
		return -1
	}

	userTokenBalance, ok := ctx.Balances[userTokenAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Swap] 缺失 user token 余额: account=%s (userTokenAccount), tx=%s", userTokenAccount, ctx.TxHashString())
		return -1
	}

	// 9. 校验 token 是否一致
	if userTokenBalance.Token != pairTokenBalance.Token {
		logger.Errorf("[Pumpfun:Swap] token 不一致: user=%s, pair=%s, tx=%s",
			userTokenBalance.Token, pairTokenBalance.Token, ctx.TxHashString())
		return -1
	}

	// 10. 校验 token 精度一致性
	if userTokenBalance.Decimals != pairTokenBalance.Decimals {
		logger.Errorf("[Pumpfun:Swap] token 精度不一致: token=%s, user.decimals=%d, pair.decimals=%d, tx=%s",
			userTokenBalance.Token, userTokenBalance.Decimals, pairTokenBalance.Decimals,
			ctx.TxHashString())
		return -1
	}

	// 11. 构建事件
	tradeEvent := &pb.TradeEvent{
		Type:      eventType,                                                           // 事件类型（BUY / SELL）
		EventId:   core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex), // 全局唯一事件 ID
		Slot:      ctx.Slot,
		BlockTime: ctx.BlockTime,
		TxHash:    ctx.TxHash,
		Signers:   ctx.Signers,
		Dex:       consts.DexPumpfun, // DEX 来源标识（Pump.fun）

		TokenDecimals: uint32(userTokenBalance.Decimals), // 交易 token 的精度
		QuoteDecimals: 9,                                 // SOL 精度固定为 9

		TokenAmount:      event.TokenAmount, // 实际成交 token 数量
		QuoteTokenAmount: event.SolAmount,   // 实际成交 SOL 数量（作为 quote）

		Token:      event.Mint[:],     // 被交易的 token（base token）
		QuoteToken: consts.SOLMint[:], // quote token（SOL）

		PairAddress:       pairAddress[:],      // 池子账户（Bonding Curve 主账户）
		TokenAccount:      pairTokenAccount[:], // 池子持有 token 的 SPL TokenAccount
		QuoteTokenAccount: pairAddress[:],      // 对于 SOL 作为 quote，无独立 token account，直接使用主账户

		UserWallet: userWallet[:], // 用户主钱包地址

		PairTokenBalance: pairTokenBalance.PostBalance, // 交易后池子的 token 余额
		PairQuoteBalance: pairSolBalance.PostBalance,   // 交易后池子的 quote 余额（SOL）

		UserTokenBalance: userTokenBalance.PostBalance, // 交易后用户的 token 余额
		UserQuoteBalance: userSolBalance.PostBalance,   // 交易后用户的 quote 余额（SOL）
	}

	// 12. 将 pair 的 SOL 余额补充为标准 token balance，统一参与后续的余额与估值处理
	ctx.Tx.AppendSolToTokenBalances(pairSolBalance)

	// 13. 添加事件到上下文
	ctx.AddEvent(&core.Event{
		ID:        tradeEvent.EventId,
		EventType: uint32(tradeEvent.Type),
		Key:       tradeEvent.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Trade{Trade: tradeEvent},
		},
	})
	return eventIndex + 1
}
