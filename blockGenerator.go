// Copyright © 2018 J. Strobus White.
// This file is part of the blocktop blockchain development kit.
//
// Blocktop is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Blocktop is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with blocktop. If not, see <http://www.gnu.org/licenses/>.

package luckyblock

import (
	"github.com/spf13/viper"
	"sync"

	spec "github.com/blckit/go-spec"
	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

type BlockGenerator struct {
	sync.Mutex
	outstandingTxns map[string]spec.Transaction
	peerID          string
	blockType				string
}

func NewBlockGenerator(peerID string) *BlockGenerator {
	g := &BlockGenerator{}
	g.blockType = viper.GetString("blockchain.block.type")
	g.outstandingTxns = make(map[string]spec.Transaction, 0)
	g.peerID = peerID
	return g
}

func (g *BlockGenerator) GetType() string {
	return g.blockType
}

func (g *BlockGenerator) Unmarshal(message proto.Message, txnHandlers map[string]spec.TransactionHandler) spec.Block {
	a, ok := message.(*any.Any)
	var msg *BlockMessage
	if ok {
		msg = &BlockMessage{}
		ptypes.UnmarshalAny(a, msg)
	} else {
		msg = message.(*BlockMessage)
	}	
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
