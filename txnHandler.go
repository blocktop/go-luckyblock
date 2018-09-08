package luckyblock

import (
	spec "github.com/blckit/go-spec"
	proto "github.com/golang/protobuf/proto"
)

type TransactionHandler struct {
	Protocol string
}

func (h *TransactionHandler) GetType() string {
	return "exchange"
}

func (h *TransactionHandler) Unmarshal(message proto.Message) spec.Transaction {
	return nil //TODO
}

func (h *TransactionHandler) Execute(txn *spec.Transaction) (ok bool) {
	//TODO
	return true
}
