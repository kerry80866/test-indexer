package pumpfun

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/internal/tools"
	"dex-indexer-sol/pb"
	"github.com/near/borsh-go"
	"runtime/debug"
)

type PumpMigrateEvent struct {
	Sign             uint64
	User             types.Pubkey
	Mint             types.Pubkey
	MintAmount       uint64
	SolAmount        uint64
	PoolMigrationFee uint64
	BondingCurve     types.Pubkey
	Timestamp        uint64
	Pool             types.Pubkey
}

// 示例交易：https://solscan.io/tx/467951uo4iDeWGtNxe8ZMGWiyeJTK1Dx2WGrK4gDNwcvgaUW9Lg4T84HRtT2d33ZRkZSZUUrihjc5ju7kZ7X7KeA
//
// Pump.fun - Migrate 指令账户布局：
//
// #0  - Global Config                         // Pump.fun 全局配置
// #1  - Withdraw Authority                    // Migration 操作发起者（Signer + Fee Payer）
// #2  - Base Token Mint                       // 被迁移的项目方 token（如 Minions）
// #3  - Bonding Curve                         // Bonding Curve 状态账户（定价逻辑）
// #4  - Bonding Curve Vault                   // 储存项目方 token 的 Vault（关联 Mint）
// #5  - User                                  // 用户钱包地址（Signer + Fee Payer）
// #6  - System Program                        // 系统程序（创建账户）
// #7  - Token Program                         // SPL Token 标准程序（Token v1）
// #8  - Pump AMM Program                      // Pump.fun AMM 主程序
// #9  - Pool                                  // AMM 池子主账户（Base-Quote 对）
// #10 - Pool Authority                        // 池子的 Authority PDA（由程序派生）
// #11 - Pool Authority Base Token Account     // Authority 持有的 base token 账户
// #12 - Pool Authority Quote Token Account    // Authority 持有的 quote token（WSOL）账户
// #13 - AMM Global Config                     // AMM 配置账户（如手续费设置）
// #14 - Quote Token Mint                      // Quote Token 的 mint（通常为 WSOL）
// #15 - LP Mint                               // LP Token 的 mint（代表流动性份额）
// #16 - User LP Token Account                 // 用户 LP token 接收账户
// #17 - Pool Base Token Account               // AMM 池中 base token 的持仓账户
// #18 - Pool Quote Token Account              // AMM 池中 quote token（WSOL）的持仓账户
// #19 - Token 2022 Program                    // 用于兼容新版 SPL Token（v2022）
// #20 - Associated Token Program              // ATA 创建与管理程序
// #21 - AMM Event Authority                   // AMM 的事件授权 PDA（事件合法性检查）
// #22 - Event Authority                       // 主 Program 的事件 PDA（权限验证）
// #23 - Pump.fun Program                      // 主程序地址（Pump.fun）
func extractMigrateEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (next int) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Pumpfun:Migrate] panic: %v, stack=%s, tx=%s", r, debug.Stack(), ctx.TxHashString())
			next = -1
		}
	}()

	ix := instrs[current]

	// 1. 校验指令结构
	if len(ix.Accounts) < 24 {
		logger.Errorf("[Pumpfun:Migrate] 指令账户长度不足: got=%d, expect>=24, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	// 2. 提取并解析事件
	eventIndex := findEventInstruction(instrs, current, ix.Accounts[22]) // Event Authority
	if eventIndex < 0 {
		// 加上预期 EventAuthority
		logger.Infof("[Pumpfun:Migrate] 未找到事件日志指令: authority=%s, tx=%s", ix.Accounts[22], ctx.TxHashString())
		return -1
	}
	eventIx := instrs[eventIndex]
	event := PumpMigrateEvent{}
	if err := borsh.Deserialize(&event, eventIx.Data[8:]); err != nil {
		logger.Errorf("[Pumpfun:Migrate] 事件反序列化失败: err=%v, dataLen=%d, tx=%s", err, len(eventIx.Data), ctx.TxHashString())
		return -1
	}

	// 3. 校验交易 token mint
	if event.Mint != ix.Accounts[2] {
		logger.Errorf("[Pumpfun:Migrate] base mint 不匹配 (expected=%s, got=%s): tx=%s", ix.Accounts[2], event.Mint, ctx.TxHashString())
		return -1
	}

	// 4. 校验BondingCurve地址一致性
	if event.BondingCurve != ix.Accounts[3] {
		logger.Errorf("[Pumpfun:Migrate] BondingCurve 不一致 (expected=%s, got=%s): tx=%s", ix.Accounts[3], event.BondingCurve, ctx.TxHashString())
		return -1
	}

	// 5. 校验用户地址一致性
	if event.User != ix.Accounts[5] {
		logger.Errorf("[Pumpfun:Migrate] 用户地址不一致 (expected=%s, got=%s): tx=%s", ix.Accounts[5], event.User, ctx.TxHashString())
		return -1
	}

	// 6. 校验 Token Program 是否为 SPL Token
	if !tools.IsSPLTokenPubkey(ix.Accounts[7]) {
		logger.Errorf("[Pumpfun:Migrate] Token Program 非 SPL 标准程序: got=%s, tx=%s", ix.Accounts[7], ctx.TxHashString())
		return -1
	}

	// 7. 提取关键账户
	srcPoolAddress := ix.Accounts[3]        // 来源池主账户（Bonding Curve）
	srcPoolTokenAccount := ix.Accounts[4]   // 来源池 base token 存储账户
	destPoolAddress := ix.Accounts[9]       // 目标池主账户（AMM Pool）
	destPoolCreator := ix.Accounts[10]      // 目标池创建者（Pool Authority）
	destPoolTokenAccount := ix.Accounts[17] // 目标池 base token 账户
	destPoolQuoteAccount := ix.Accounts[18] // 目标池 quote token 账户（如 WSOL）

	// 8. 获取来源池 base token 余额
	srcPoolTokenBalance, ok := ctx.Balances[srcPoolTokenAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Migrate] 来源池 base token 余额缺失: account=%s, tx=%s", srcPoolTokenAccount, ctx.TxHashString())
		return -1
	}
	// 校验 base token account 所有者是否为来源池地址
	if srcPoolTokenBalance.PostOwner != srcPoolAddress {
		logger.Errorf("[Pumpfun:Migrate] 来源池 base token 账户所有者异常: expected=%s, actual=%s, account=%s, tx=%s",
			srcPoolAddress, srcPoolTokenBalance.PostOwner, srcPoolTokenAccount, ctx.TxHashString())
		return -1
	}

	// 9. 获取来源池 SOL 余额
	srcPoolSolBalance, ok := ctx.Tx.SolBalances[srcPoolAddress]
	if !ok {
		logger.Errorf("[Pumpfun:Migrate] 来源池 SOL 余额缺失: account=%s, tx=%s", srcPoolAddress, ctx.TxHashString())
		return -1
	}

	// 10. 获取目标池 base token 余额
	destPoolTokenBalance, ok := ctx.Balances[destPoolTokenAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Migrate] 目标池 base token 余额缺失: account=%s, tx=%s", destPoolTokenAccount, ctx.TxHashString())
		return -1
	}

	// 11. 获取目标池 quote token 余额
	destPoolQuoteBalance, ok := ctx.Balances[destPoolQuoteAccount]
	if !ok {
		logger.Errorf("[Pumpfun:Migrate] 目标池 quote token 余额缺失: account=%s, tx=%s", destPoolQuoteAccount, ctx.TxHashString())
		return -1
	}

	// 12. 校验 base token 一致性（来源池、目标池与事件中的 mint 必须相同）
	if srcPoolTokenBalance.Token != destPoolTokenBalance.Token || event.Mint != srcPoolTokenBalance.Token {
		logger.Errorf("[Pumpfun:Migrate] base token 不一致: src=%s, dest=%s, event=%s, tx=%s",
			srcPoolTokenBalance.Token, destPoolTokenBalance.Token, event.Mint, ctx.TxHashString())
		return -1
	}

	// 13. 校验 base token 的精度一致性（可选）
	if srcPoolTokenBalance.Decimals != destPoolTokenBalance.Decimals {
		logger.Errorf("[Pumpfun:Migrate] base token 精度不一致: src=%d, dest=%d, tx=%s",
			srcPoolTokenBalance.Decimals, destPoolTokenBalance.Decimals, ctx.TxHashString())
		return -1
	}

	// 14. 校验 quote token 是否为 WSOL
	if destPoolQuoteBalance.Token != consts.WSOLMint {
		logger.Errorf("[Pumpfun:Migrate] quote token 类型错误: expected=%s, actual=%s, tx=%s",
			consts.WSOLMint, destPoolQuoteBalance.Token, ctx.TxHashString())
		return -1
	}

	// 15. 校验 quote token 精度是否为 9
	if destPoolQuoteBalance.Decimals != 9 {
		logger.Errorf("[Pumpfun:Migrate] quote token 精度错误: expected=9, actual=%d, tx=%s",
			destPoolQuoteBalance.Decimals, ctx.TxHashString())
		return -1
	}

	// 16. 将池子的 SOL 余额补充进 token balance 列表，便于后续统一处理
	ctx.Tx.AppendSolToTokenBalances(srcPoolSolBalance)

	// 17. 构建MigrateEvent
	migrateEvent := &pb.MigrateEvent{
		Type:      pb.EventType_MIGRATE,
		EventId:   core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:      ctx.Slot,
		BlockTime: ctx.BlockTime,
		TxHash:    ctx.TxHash,
		Signers:   ctx.Signers,

		UserWallet:      event.User[:],
		DestPoolCreator: destPoolCreator[:],

		SrcDex:  consts.DexPumpfun,
		DestDex: consts.DexPumpfunAMM,

		TokenDecimals: uint32(destPoolTokenBalance.Decimals),
		QuoteDecimals: uint32(destPoolQuoteBalance.Decimals),

		TokenAmount:      event.MintAmount,       // 迁移base token数量
		QuoteTokenAmount: event.SolAmount,        // 迁移SOL数量
		MigrationFee:     event.PoolMigrationFee, // 迁移费用(SOL)

		Token:          event.Mint[:],
		SrcQuoteToken:  consts.SOLMint[:],
		DestQuoteToken: destPoolQuoteBalance.Token[:],

		SrcPairAddress:  srcPoolAddress[:],
		DestPairAddress: destPoolAddress[:],

		SrcTokenAccount:  srcPoolTokenAccount[:],
		DestTokenAccount: destPoolTokenAccount[:],

		// 对于 SOL，没有 token account，直接使用主账户地址。
		SrcQuoteTokenAccount:  srcPoolAddress[:],
		DestQuoteTokenAccount: destPoolQuoteAccount[:],

		SrcTokenAccountOwner:  srcPoolTokenBalance.PostOwner[:],
		DestTokenAccountOwner: destPoolTokenBalance.PostOwner[:],

		// 对于 SOL 主账户，其 owner 是自己
		SrcQuoteTokenAccountOwner:  srcPoolAddress[:],
		DestQuoteTokenAccountOwner: destPoolQuoteBalance.PostOwner[:],

		SrcPairTokenBalance:  srcPoolTokenBalance.PostBalance,
		DestPairTokenBalance: destPoolTokenBalance.PostBalance,
		SrcPairQuoteBalance:  srcPoolSolBalance.PostBalance,
		DestPairQuoteBalance: destPoolQuoteBalance.PostBalance,
	}

	// 18. 添加事件到事件列表
	ctx.AddEvent(&core.Event{
		ID:        migrateEvent.EventId,
		EventType: uint32(migrateEvent.Type),
		Key:       migrateEvent.SrcPairAddress, // 分区 Key 用旧池 SrcPairAddress 更符合事件语义
		Event: &pb.Event{
			Event: &pb.Event_Migrate{Migrate: migrateEvent},
		},
	})

	// 保留内部的新池事件
	return current + 1
}
