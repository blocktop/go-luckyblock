package luckyblock

import (
	"sync"

	spec "github.com/blckit/go-spec"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

type BlockGenerator struct {
	sync.Mutex
	outstandingTxns map[string]spec.Transaction
	peerID          string
}

func NewBlockGenerator(peerID string) *BlockGenerator {
	g := &BlockGenerator{}
	g.outstandingTxns = make(map[string]spec.Transaction, 0)
	g.peerID = peerID
	return g
}

func (g *BlockGenerator) GetType() string {
	return BlockType
}

func (g *BlockGenerator) Unmarshal(message proto.Message, txnHandlers map[string]spec.TransactionHandler) spec.Block {
	msg := &BlockMessage{}
	ptypes.UnmarshalAny(message.(*any.Any), msg)

	block := &Block{}
	block.Unmarshal(msg, txnHandlers)

	return block
}

func (g *BlockGenerator) LogTransaction(txn spec.Transaction) {
	g.Lock()
	g.outstandingTxns[txn.GetID()] = txn
	g.Unlock()
}

func (g *BlockGenerator) UnlogTransactions(txns []spec.Transaction) {
	g.Lock()
	for _, t := range txns {
		delete(g.outstandingTxns, t.GetID())
	}
	g.Unlock()
}

func (g *BlockGenerator) ProduceGenesisBlock() spec.Block {
	block := NewBlock(nil, g.peerID)
	block.GenerateID()
	return block
}
func (g *BlockGenerator) GenerateBlock(branch []spec.Block) (newBlock spec.Block) {
	// do the work, generate block
	// in the case of luckyblock there is no work, blocks are evaluated by their score
	head := branch[0].(*Block)
	block := NewBlock(head, g.peerID)

	branchtxns := make([]spec.Transaction, 0)
	for _, b := range branch {
		branchtxns = append(branchtxns, b.GetTransactions()...)
	}

	// make list of outstanding transactions that are not already in blocks of this branch
	newTxns := make([]spec.Transaction, 0)
	g.Lock()
	for id, t := range g.outstandingTxns {
		if !containsTransaction(branchtxns, id) {
			newTxns = append(newTxns, t)
		}
	}
	g.Unlock()

	block.transactions = newTxns

	block.GenerateID()

	return block

}

func containsTransaction(txns []spec.Transaction, id string) bool {
	for _, t := range txns {
		if t.GetID() == id {
			return true
		}
	}
	return false
}
