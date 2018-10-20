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
	"sync"

	"github.com/blocktop/go-kernel"

	"github.com/golang/glog"
	"github.com/spf13/viper"

	spec "github.com/blocktop/go-spec"
)

type BlockGenerator struct {
	sync.Mutex
	outstandingTxns map[string]spec.Transaction
	prID            string
	blockType       string
	txnHandlers     map[string]spec.TransactionHandler
	blockCommitter  *blockCommitter
}

var _ spec.BlockGenerator = (*BlockGenerator)(nil)

func NewBlockGenerator(txnHandlers ...spec.TransactionHandler) *BlockGenerator {
	g := &BlockGenerator{}
	g.blockType = viper.GetString("blockchain.block.type")
	g.outstandingTxns = make(map[string]spec.Transaction, 0)

	g.txnHandlers = make(map[string]spec.TransactionHandler, len(txnHandlers))
	for _, h := range txnHandlers {
		g.txnHandlers[h.Type()] = h
	}

	g.blockCommitter = &blockCommitter{}

	return g
}

func (g *BlockGenerator) peerID() string {
	if g.prID == "" {
		g.prID = kernel.Network().PeerID()
	}
	return g.prID
}

func (g *BlockGenerator) logPeerID() string {
	prID := g.peerID()
	if prID[:2] == "Qm" {
		return prID[2:8]
	}
	return prID[:6]
}

func (g *BlockGenerator) Type() string {
	return g.blockType
}

func (g *BlockGenerator) BlockPrototype() spec.Block {
	return NewBlock(nil, "")
}

func (g *BlockGenerator) GenerateGenesisBlock() spec.Block {
	block := NewBlock(nil, g.peerID())
	return block
}
func (g *BlockGenerator) GenerateBlock(branch []spec.Block) (newBlock spec.Block) {
	// do the work, generate block
	// in the case of luckyblock there is no work, blocks are evaluated by their score
	head := branch[0].(*Block)
	block := NewBlock(head, g.peerID())

	branchtxns := make([]spec.Transaction, 0)
	for _, b := range branch {
		branchtxns = append(branchtxns, b.Transactions()...)
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

	block.txns = newTxns

	return block

}

func (g *BlockGenerator) ReceiveTransaction(netMsg *spec.NetworkMessage) (spec.Transaction, error) {
	txnType := netMsg.Protocol.ResourceType()
	h := g.txnHandlers[txnType]
	if h == nil {
		glog.Warningf("%s received transaction of unknown type: %s", g.blockType, txnType)
		return nil, nil
	}
	txn, err := h.ReceiveTransaction(netMsg)
	if err != nil {
		return nil, err
	}
	g.logTransaction(txn)

	return txn, nil
}

func (g *BlockGenerator) TryCommitBlock(newBlock spec.Block, branch []spec.Block) bool {
	//_, err := g.blockCommitter.TryCommit(newBlock, branch)
	//return err != nil
	return true
}

func (g *BlockGenerator) CommitBlock(block spec.Block) {
	//go g.blockCommitter.Commit(block)
}

func (g *BlockGenerator) executeTransactions(txns []spec.Transaction) bool {
	for _, t := range txns {
		txnType := t.Name()
		handler := g.txnHandlers[txnType]
		if handler == nil {
			// TODO: log something here
			// if we can't confirm txn then our data will be corrupt
			// or no one else will be able to either
			// or could be security issue
			return false
		} else {
			if !handler.Execute(t) {
				//TODO log and fail
				return false
			}
		}
	}
	return true
}

func (g *BlockGenerator) logTransaction(txn spec.Transaction) {
	g.Lock()
	g.outstandingTxns[txn.Hash()] = txn
	g.Unlock()
}

func (g *BlockGenerator) unlogTransactions(txns []spec.Transaction) {
	g.Lock()
	for _, t := range txns {
		delete(g.outstandingTxns, t.Hash())
	}
	g.Unlock()
}

func containsTransaction(txns []spec.Transaction, id string) bool {
	for _, t := range txns {
		if t.Hash() == id {
			return true
		}
	}
	return false
}
