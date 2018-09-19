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

	"github.com/golang/glog"
	"github.com/spf13/viper"

	spec "github.com/blocktop/go-spec"
	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

type BlockGenerator struct {
	sync.Mutex
	outstandingTxns map[string]spec.Transaction
	peerID          string
	logPeerID       string
	blockType       string
	txnHandlers     map[string]spec.TransactionHandler
}

func NewBlockGenerator(peerID string, txnHandlers ...spec.TransactionHandler) *BlockGenerator {
	g := &BlockGenerator{}
	g.blockType = viper.GetString("blockchain.block.type")
	g.outstandingTxns = make(map[string]spec.Transaction, 0)
	g.peerID = peerID

	if peerID[:2] == "Qm" {
		g.logPeerID = peerID[2:8]
	} else {
		g.logPeerID = peerID[:6]
	}

	g.txnHandlers = make(map[string]spec.TransactionHandler, len(txnHandlers))
	for _, h := range txnHandlers {
		g.txnHandlers[h.GetType()] = h
	}

	return g
}

func (g *BlockGenerator) GetType() string {
	return g.blockType
}

func (g *BlockGenerator) GenerateGenesisBlock() spec.Block {
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

func (g *BlockGenerator) ReceiveTransaction(netMsg *spec.NetworkMessage) {
	txnType := netMsg.Protocol.GetResourceType()
	h := g.txnHandlers[txnType]
	if h == nil {
		glog.Warningf("Peer %s: %s received transaction of unknown type: %s", g.logPeerID, g.blockType, txnType)
		return
	}
	txn := h.Unmarshal(netMsg.Message)
	g.logTransaction(txn)

}

func (g *BlockGenerator) CommitBlock(block spec.Block) {
	txns := block.GetTransactions()
	if !g.executeTransactions(txns) {
		// TODO fail to commit
		// may want to unlog the problem transaction(s) here
		return
	}

	g.unlogTransactions(txns)

	glog.Warningf("Peer %s: %s confirmed block %d: %s", g.logPeerID, g.blockType, block.GetBlockNumber(), block.GetID()[:6])
}

func (g *BlockGenerator) Unmarshal(message proto.Message) spec.Block {
	a, ok := message.(*any.Any)
	var msg *BlockMessage
	if ok {
		msg = &BlockMessage{}
		ptypes.UnmarshalAny(a, msg)
	} else {
		msg = message.(*BlockMessage)
	}

	txnMsgs := msg.GetTransactions()
	txns := make([]spec.Transaction, len(txnMsgs))
	for i, any := range txnMsgs {
		txnType, err := ptypes.AnyMessageName(any)
		if err != nil {
			//TODO
			continue
		}
		h := g.txnHandlers[txnType]
		if h == nil {
			// TODO, how to handle
			continue
		}
		txns[i] = h.Unmarshal(any)
	}

	block := &Block{}

	block.Unmarshal(msg, txns)

	return block
}

func (g *BlockGenerator) executeTransactions(txns []spec.Transaction) bool {
	for _, t := range txns {
		txnType := t.GetType()
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
	g.outstandingTxns[txn.GetID()] = txn
	g.Unlock()
}

func (g *BlockGenerator) unlogTransactions(txns []spec.Transaction) {
	g.Lock()
	for _, t := range txns {
		delete(g.outstandingTxns, t.GetID())
	}
	g.Unlock()
}

func containsTransaction(txns []spec.Transaction, id string) bool {
	for _, t := range txns {
		if t.GetID() == id {
			return true
		}
	}
	return false
}
