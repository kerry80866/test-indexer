package txadapter

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"fmt"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

// buildFullAccountKeys 构造完整账户列表，合并 message.accountKeys 和 Address Lookup Table 地址，便于索引。
func buildFullAccountKeys(
	accountKeys, loadedWritable, loadedReadonly [][]byte,
) ([]types.Pubkey, error) {
	// 计算总账户数，确保分配空间恰好
	total := len(accountKeys) + len(loadedWritable) + len(loadedReadonly)
	pubkeys := make([]types.Pubkey, total)

	i := 0 // 写入索引

	// 主账户部分（来自 message.accountKeys）
	for _, b := range accountKeys {
		if len(b) != 32 {
			return nil, fmt.Errorf("invalid pubkey in accountKeys at index %d", i)
		}
		copy(pubkeys[i][:], b)
		i++
	}

	// Address Table 中的 writable 部分
	for _, b := range loadedWritable {
		if len(b) != 32 {
			return nil, fmt.Errorf("invalid pubkey in loadedWritable at index %d", i)
		}
		copy(pubkeys[i][:], b)
		i++
	}

	// Address Table 中的 readonly 部分
	for _, b := range loadedReadonly {
		if len(b) != 32 {
			return nil, fmt.Errorf("invalid pubkey in loadedReadonly at index %d", i)
		}
		copy(pubkeys[i][:], b)
		i++
	}
	return pubkeys, nil
}

// buildSolBalances 构建交易的 SOL 余额变化映射（包含账户、余额前后快照、索引等）。
func buildSolBalances(
	tx *pb.SubscribeUpdateTransactionInfo,
	accountKeys []types.Pubkey,
) map[types.Pubkey]*core.SolBalance {
	preList := tx.Meta.PreBalances   // 交易执行前余额列表
	postList := tx.Meta.PostBalances // 交易执行后余额列表

	balanceMap := make(map[types.Pubkey]*core.SolBalance, len(preList)+len(postList))

	for i, preBalance := range preList {
		account := accountKeys[i]
		balanceMap[account] = &core.SolBalance{
			Account:     account,
			PreBalance:  preBalance,
			PostBalance: 0,
			TxIndex:     uint16(tx.Index),
			InnerIndex:  uint16(i),
		}
	}

	// 再补充 Post，如果账户存在则更新，否则新增
	newIndex := len(preList)
	for i, postBalance := range postList {
		account := accountKeys[i]
		if tb, ok := balanceMap[account]; ok {
			tb.PostBalance = postBalance
		} else {
			balanceMap[account] = &core.SolBalance{
				Account:     account,
				PreBalance:  0,
				PostBalance: postBalance,
				TxIndex:     uint16(tx.Index),
				InnerIndex:  uint16(newIndex),
			}
			newIndex++
		}
	}

	return balanceMap
}

// buildBalances 构建交易中的 Token 余额变化及 mint → decimals 映射（含去重与所有权信息）。
// 返回：
//   - balanceMap：token account → TokenBalance（含 mint、owner、pre/post 余额等）
//   - tokenDecimals：当前交易中涉及的 mint → decimals（去重 + 有序）
func buildBalances(
	owners map[string]types.Pubkey,
	tx *pb.SubscribeUpdateTransactionInfo,
	accountKeys []types.Pubkey,
) (map[types.Pubkey]*core.TokenBalance, []core.TokenDecimals) {
	preList := tx.Meta.PreTokenBalances
	postList := tx.Meta.PostTokenBalances

	// token account 数量预估，用于预分配 map 和 mintResolver 缓存
	capacity := len(preList) + len(postList)

	// 初始化余额映射
	balanceMap := make(map[types.Pubkey]*core.TokenBalance, capacity)

	// mintResolver：用于构建 mint → decimals 的映射，避免重复记录 decimals 信息。
	mintResolver := newMintResolver(capacity)

	// ownerResolver：用于将 base58 owner 地址解析为 Pubkey，并缓存解码结果。
	ownerResolver := newOwnerResolver(owners)

	// 先处理 Post（代表账户最终状态），初始化结构，PreBalance 默认为 0
	for i, post := range postList {
		// 仅处理标准 SPL Token（TokenProgram / Token2022），跳过非标准模拟账户
		if utils.IsSPLToken(post.ProgramId) {
			account := accountKeys[post.AccountIndex]
			decimals := uint8(post.UiTokenAmount.Decimals)
			balanceMap[account] = &core.TokenBalance{
				TokenAccount: account,
				Token:        mintResolver.resolve(post.Mint, decimals),
				PostBalance:  utils.ParseUint64(post.UiTokenAmount.Amount),
				PostOwner:    ownerResolver.resolve(post.Owner),
				Decimals:     decimals,
				TxIndex:      uint16(tx.Index),
				InnerIndex:   uint16(i),
			}
		}
	}

	// 再补充 Pre（如账户只出现在 Pre 中，说明可能被销毁）
	newIndex := len(preList)
	for _, pre := range preList {
		// 仅处理标准 SPL Token（TokenProgram / Token2022），跳过非标准模拟账户
		if utils.IsSPLToken(pre.ProgramId) {
			account := accountKeys[pre.AccountIndex]
			owner := ownerResolver.resolve(pre.Owner)
			if tb, ok := balanceMap[account]; ok {
				// 账户在 Post 中已存在，这里补充 PreBalance
				tb.HasPreOwner = true
				tb.PreOwner = owner
				tb.PreBalance = utils.ParseUint64(pre.UiTokenAmount.Amount)
			} else {
				// Pre-only（如销毁账户），构造最小结构
				decimals := uint8(pre.UiTokenAmount.Decimals)
				balanceMap[account] = &core.TokenBalance{
					TokenAccount: account,
					Token:        mintResolver.resolve(pre.Mint, decimals),
					HasPreOwner:  true,
					PreOwner:     owner,
					PostOwner:    owner, // Pre-only 情况默认设置 PostOwner = PreOwner
					PreBalance:   utils.ParseUint64(pre.UiTokenAmount.Amount),
					Decimals:     decimals,
					TxIndex:      uint16(tx.Index),
					InnerIndex:   uint16(newIndex),
				}
				newIndex++
			}
		}
	}

	return balanceMap, mintResolver.buildTokenDecimals()
}

// buildInstructions 扁平化主指令与 inner 指令，生成统一的 AdaptedInstruction 列表。
func buildInstructions(
	tx *pb.SubscribeUpdateTransactionInfo,
	accountKeys []types.Pubkey,
) []*core.AdaptedInstruction {
	rawInstructions := tx.Transaction.Message.Instructions
	rawInners := tx.Meta.InnerInstructions

	// 预分配容量：假设每条主指令平均含有 2 条 inner 指令，最低保留 32 条，避免切片动态扩容
	instructions := make([]*core.AdaptedInstruction, 0, utils.Max(len(rawInstructions)*2, 32))
	innerIndex := 0

	for i, inst := range rawInstructions {
		// 解析主指令，标记 InnerIndex = 0
		accounts := make([]types.Pubkey, 0, len(inst.Accounts))
		for _, idx := range inst.Accounts {
			accounts = append(accounts, accountKeys[idx])
		}
		instructions = append(instructions, &core.AdaptedInstruction{
			IxIndex:    uint16(i),
			InnerIndex: 0,
			ProgramID:  accountKeys[inst.ProgramIdIndex],
			Accounts:   accounts,
			Data:       inst.Data,
		})

		// 解析 inner 指令（如存在），InnerIndex 从1开始递增
		// 注意：Solana 标准中，每个主指令最多对应一个 inner 指令块，
		// 且 inner 列表按主指令索引（Index）递增排列，因此此处采用顺序匹配，无需 map 或多次扫描。
		if innerIndex < len(rawInners) && int(rawInners[innerIndex].Index) == i {
			for j, inner := range rawInners[innerIndex].Instructions {
				innerAccounts := make([]types.Pubkey, 0, len(inner.Accounts))
				for _, idx := range inner.Accounts {
					innerAccounts = append(innerAccounts, accountKeys[idx])
				}
				instructions = append(instructions, &core.AdaptedInstruction{
					IxIndex:    uint16(i),
					InnerIndex: uint16(j + 1), // InnerIndex从1开始递增
					ProgramID:  accountKeys[inner.ProgramIdIndex],
					Accounts:   innerAccounts,
					Data:       inner.Data,
				})
			}
			innerIndex++
		}
	}

	return instructions
}

// AdaptGrpcTx 将 gRPC 推送的交易数据解析为内部 AdaptedTx 结构。
// 完整流程：
//  1. 构建 accountKeys（含 Address Lookup）；
//  2. 构建指令（主 + inner）；
//  3. 构建 Token 余额（含 decimals 去重）；
//  4. 返回 AdaptedTx；如 panic 会被 recover。
//
// owners：goroutine 私有的 Base58 → Pubkey 缓存，不会跨协程共享
func AdaptGrpcTx(txCtx *core.TxContext, owners map[string]types.Pubkey, tx *pb.SubscribeUpdateTransactionInfo) (_ *core.AdaptedTx, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("AdaptGrpcTx panic: %v", r)
		}
	}()

	// 构造完整的账户 pubkey 列表（主账户 + Address Lookup 表中的 writable 和 readonly）
	accountKeys, err := buildFullAccountKeys(
		tx.Transaction.Message.AccountKeys,
		tx.Meta.LoadedWritableAddresses,
		tx.Meta.LoadedReadonlyAddresses,
	)
	if err != nil {
		return nil, fmt.Errorf("buildFullAccountKeys error: %w", err)
	}

	// 基本健壮性校验：签名或账户列表为空时立即报错
	if len(tx.Transaction.Signatures) == 0 || len(accountKeys) == 0 {
		return nil, fmt.Errorf("invalid transaction: missing signature or accountKeys")
	}

	// 获取 signer 数量（前 N 个 accountKeys 视为 signer）
	signerCount := int(tx.Transaction.Message.Header.NumRequiredSignatures)
	if signerCount == 0 || len(accountKeys) < signerCount {
		return nil, fmt.Errorf("invalid signer count: %d", signerCount)
	}

	// 解析主指令和 inner 指令
	instructions := buildInstructions(tx, accountKeys)

	// 解析 pre/post token 余额 + decimals 信息
	balances, tokenDecimals := buildBalances(owners, tx, accountKeys)

	// 构造签名者列表：Solana 中交易前 N 个账户即为 signer
	signers := make([][]byte, signerCount)
	for i := 0; i < signerCount; i++ {
		signers[i] = accountKeys[i][:]
	}

	// 组装最终结构体
	return &core.AdaptedTx{
		TxCtx:         txCtx,
		TxIndex:       uint32(tx.Index),
		Signature:     tx.Transaction.Signatures[0],
		Signers:       signers,
		Instructions:  instructions,
		SolBalances:   buildSolBalances(tx, accountKeys),
		Balances:      balances,
		TokenDecimals: tokenDecimals,
	}, nil
}
