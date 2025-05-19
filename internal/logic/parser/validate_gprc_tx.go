package parser

import (
	"fmt"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

func ValidateGrpcTx(tx *pb.SubscribeUpdateTransactionInfo) error {
	if tx == nil {
		return fmt.Errorf("nil transaction info")
	}
	if tx.Transaction == nil {
		return fmt.Errorf("missing Transaction field")
	}
	if tx.Transaction.Message == nil {
		return fmt.Errorf("missing Message field in transaction")
	}
	if len(tx.Transaction.Signatures) == 0 {
		return fmt.Errorf("missing transaction signature")
	}
	if len(tx.Transaction.Signatures[0]) != 64 {
		return fmt.Errorf("invalid transaction signature length: %d", len(tx.Transaction.Signatures[0]))
	}
	if tx.IsVote {
		return fmt.Errorf("vote transaction skipped")
	}
	if tx.Meta == nil {
		return fmt.Errorf("missing transaction meta data")
	}
	if tx.Meta.Err != nil {
		return fmt.Errorf("transaction execution failed: %v", tx.Meta.Err)
	}
	return nil
}
