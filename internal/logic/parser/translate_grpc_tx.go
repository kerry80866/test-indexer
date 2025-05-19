package parser

import (
	"dex-indexer-sol/internal/logic/domain"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"fmt"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

// buildFullAccountKeys 构造交易中完整的账户 Pubkey 列表。
// 将 message.accountKeys 和 Address Lookup Table 中的 writable / readonly 地址
// 顺序拼接为一个 []Pubkey 切片，供后续通过 accountIndex 高效索引。
//
// 性能设计说明：
//   - 预计算总长度，一次性分配目标切片，避免 append 扩容；
//   - 使用单一索引顺序写入，避免 slice 操作带来的额外开销；
//   - 全程顺序写入 + copy，有助于 CPU cache 命中。
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

// buildTranslatedBalances 构建交易中的 Token 余额信息与 decimals 精度映射。
// 从 Pre/PostTokenBalances 中提取标准 SPL Token（TokenProgram / Token-2022）账户的余额状态，
// 并记录每个 mint 对应的 decimals（去重 + 包含系统 token）。
//
// 返回：
// - balanceMap：以 token account 为 key 的 TokenBalance 映射（含 mint、owner、pre/post 余额）
// - tokenDecimals：当前交易中涉及的 mint → decimals 映射（含系统 token）
func buildTranslatedBalances(
	tx *pb.SubscribeUpdateTransactionInfo,
	accountKeys []types.Pubkey,
) (map[types.Pubkey]*domain.TokenBalance, []domain.TokenDecimals) {
	postList := tx.Meta.PostTokenBalances
	preList := tx.Meta.PreTokenBalances

	// token account 数量预估，用于预分配 map 和 resolver 缓存
	capacity := len(preList) + len(postList)

	// 初始化余额映射
	balanceMap := make(map[types.Pubkey]*domain.TokenBalance, capacity)

	// resolver：用于构建 mint → decimals 的映射，避免重复记录 decimals 信息。
	// 内部使用数组（而非 map）实现，具备以下优势：
	//   1. 每笔交易涉及的 mint 数量较少（通常 <10），数组顺序去重效率更高；
	//   2. 顺序写入，输出顺序稳定，便于展示或后续处理；
	//   3. 避免 map 带来的哈希计算与 GC 压力，在高频场景下更省内存；
	//   4. 对外接口为 []TokenDecimals，直接满足业务侧消费需求。
	resolver := NewMintResolver(capacity)

	// 先处理 Post（代表账户最终状态），初始化结构，PreBalance 默认为 0
	for _, post := range postList {
		// 仅处理标准 SPL Token（TokenProgram / Token2022），跳过非标准模拟账户
		if utils.IsSPLToken(post.ProgramId) {
			account := accountKeys[post.AccountIndex]
			decimals := uint8(post.UiTokenAmount.Decimals)
			balanceMap[account] = &domain.TokenBalance{
				TokenAccount: account,
				Token:        resolver.Resolve(post.Mint, decimals),
				Owner:        post.Owner,
				PostBalance:  utils.ParseUint64(post.UiTokenAmount.Amount),
				Decimals:     decimals,
				// PreBalance 默认为 0，稍后补充（如有）
			}
		}
	}

	// 再补充 Pre（如账户只出现在 Pre 中，说明可能被销毁）
	for _, pre := range preList {
		// 仅处理标准 SPL Token（TokenProgram / Token2022），跳过非标准模拟账户
		if utils.IsSPLToken(pre.ProgramId) {
			account := accountKeys[pre.AccountIndex]
			if tb, ok := balanceMap[account]; ok {
				// 账户在 Post 中已存在，这里补充 PreBalance
				tb.PreBalance = utils.ParseUint64(pre.UiTokenAmount.Amount)
			} else {
				// Pre-only（如销毁账户），构造最小结构
				decimals := uint8(pre.UiTokenAmount.Decimals)
				balanceMap[account] = &domain.TokenBalance{
					TokenAccount: account,
					Token:        resolver.Resolve(pre.Mint, decimals),
					Owner:        pre.Owner,
					PreBalance:   utils.ParseUint64(pre.UiTokenAmount.Amount),
					Decimals:     decimals,
					// PostBalance 默认为 0，表示账户被销毁或清空
				}
			}
		}
	}

	return balanceMap, resolver.BuildTokenDecimals()
}

// buildTranslatedInstructions 将 gRPC 推送的主指令和 inner 指令解析并转换为内部结构体列表。
// 它将原始指令 (rawInstructions) 与对应的 inner 指令 (rawInners) 逐一匹配，构建出 []*TranslatedInstruction。
func buildTranslatedInstructions(
	tx *pb.SubscribeUpdateTransactionInfo,
	accountKeys []types.Pubkey,
) []*domain.TranslatedInstruction {
	rawInstructions := tx.Transaction.Message.Instructions
	rawInners := tx.Meta.InnerInstructions

	instructions := make([]*domain.TranslatedInstruction, 0, len(rawInstructions))
	innerIndex := 0 // 顺序推进的指针，用于定位 rawInners 中尚未处理的项

	for i, inst := range rawInstructions {
		var innerInstructions []domain.Instruction

		// 如果当前 rawInner 与主指令索引 i 匹配，则解析 inner 指令列表
		if innerIndex < len(rawInners) && int(rawInners[innerIndex].Index) == i {
			innerList := rawInners[innerIndex].Instructions
			innerInstructions = make([]domain.Instruction, 0, len(innerList))
			for _, inner := range innerList {
				innerInstructions = append(innerInstructions, convertInnerInstruction(inner, accountKeys))
			}
			innerIndex++
		}

		// 构造结构化的 TranslatedInstruction（包含主指令及其 inner 指令）
		instructions = append(instructions, &domain.TranslatedInstruction{
			Instruction: convertInstruction(inst, accountKeys),
			Inners:      innerInstructions, // 若无 inner，保持 nil，避免无效内存分配
		})
	}

	return instructions
}

// convertInstruction 是 convertRawInstruction 的包装器，用于处理主指令结构（pb.CompiledInstruction）。
func convertInstruction(inst *pb.CompiledInstruction, accountKeys []types.Pubkey) domain.Instruction {
	return convertRawInstruction(inst.ProgramIdIndex, inst.Accounts, inst.Data, accountKeys)
}

// convertInnerInstruction 是 convertRawInstruction 的包装器，用于处理 inner 指令结构（pb.InnerInstruction）。
func convertInnerInstruction(inst *pb.InnerInstruction, accountKeys []types.Pubkey) domain.Instruction {
	return convertRawInstruction(inst.ProgramIdIndex, inst.Accounts, inst.Data, accountKeys)
}

// convertRawInstruction 是主解析函数，将原始字段映射为内部结构 domain.Instruction。
// accounts 字段是 accountKey 索引的 byte 列表，这里会反解为真实 Pubkey。
func convertRawInstruction(pidIdx uint32, accounts []byte, data []byte, accountKeys []types.Pubkey) domain.Instruction {
	accs := make([]types.Pubkey, 0, len(accounts))
	for _, idx := range accounts {
		accs = append(accs, accountKeys[idx])
	}
	return domain.Instruction{
		ProgramID: accountKeys[pidIdx],
		Accounts:  accs,
		Data:      data,
	}
}

// TranslateGrpcTx 解析 gRPC 推送的交易数据，并构建为内部结构 TranslatedTx。
// 包含以下处理流程：
//  1. 构建完整的 accountKeys（含 Address Lookup）
//  2. 构造指令（主指令 + inner）
//  3. 构造余额信息（Pre/Post Token Balance）
//  4. 若发生 panic，将被捕获并转为错误返回，避免程序崩溃
func TranslateGrpcTx(txCtx *domain.TxContext, tx *pb.SubscribeUpdateTransactionInfo) (_ *domain.TranslatedTx, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("TranslateGrpcTx panic: %v", r)
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
	instructions := buildTranslatedInstructions(tx, accountKeys)

	// 解析 pre/post token 余额 + decimals 信息
	balances, tokenDecimals := buildTranslatedBalances(tx, accountKeys)

	// 组装最终结构体
	return &domain.TranslatedTx{
		TxCtx:         txCtx,
		TxIndex:       tx.Index,
		Signature:     tx.Transaction.Signatures[0],
		Signer:        accountKeys[0], // 默认取第一个 signer，通常为交易发起者
		Instructions:  instructions,
		Balances:      balances,
		TokenDecimals: tokenDecimals,
	}, nil
}
