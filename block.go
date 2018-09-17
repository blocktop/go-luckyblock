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
	"crypto/sha256"
	"encoding/hex"
	"math/bits"
	"strconv"
	"time"

	"github.com/spf13/viper"

	spec "github.com/blckit/go-spec"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

const (
	MaxScore = uint64(^uint32(0))
)

type Block struct {
	id, parentID string
	blockNumber  uint64
	score        uint32
	scored       bool
	transactions []spec.Transaction
	timestamp    int64
	peerID       string
	blockType    string
	blockVersion string
}

func NewBlock(parent *Block, peerID string) *Block {
	b := &Block{}
	b.peerID = peerID
	b.timestamp = time.Now().UnixNano() / int64(time.Millisecond)
	b.blockType = viper.GetString("blockchain.block.type")
	b.blockVersion = viper.GetString("blockchain.block.version")

	if parent == nil {
		// genesis block
		b.blockNumber = uint64(0)
		b.parentID = ""
	} else {
		b.blockNumber = parent.GetBlockNumber() + uint64(1)
		b.parentID = parent.GetID()
	}

	b.transactions = make([]spec.Transaction, 0)

	return b
}

func (b *Block) GetType() string {
	return b.blockType
}

func (b *Block) GetVersion() string {
	return b.blockVersion
}

func (b *Block) GetID() string {
	return b.id
}

func (b *Block) GetParentID() string {
	return b.parentID
}

func (b *Block) GetBlockNumber() uint64 {
	return b.blockNumber
}

func (b *Block) Validate() bool {
	return true
}

func (b *Block) GetTransactions() []spec.Transaction {
	return b.transactions
}

func (b *Block) GetTimestamp() int64 {
	return b.timestamp
}

func (b *Block) GetPeerID() string {
	return b.peerID
}

func (b *Block) GetScore() uint32 {
	if !b.scored {
		score := uint64(1)
		reps := 4
		for reps > 0 {
			hash := sha256.Sum256([]byte(b.peerID + b.parentID + strconv.FormatUint(score, 16)))
			count := uint64(onesCount256(hash))
			if count == 0 {
				count++
			}
			score *= uint64(count)
			reps--
		}

		// score > MaxScore would happen if each rep yielded 256,
		// in other words if each hash rep had all 1 bits and no 0s. Thus
		// (256 * 256 * 256 * 256) == (2**8 * 2**8 * 2**8 * 2**8) == 2**32
		// which would be 1 greater than the max value of uint32. Hence
		// the check below, and the reason the scoring loop uses uint64
		// instead of uint32. This is a HIGHLY unlikely scenario. The
		// possiblity is so remote it's almost not worth the nanoseconds
		// to make this check. But just to be pedantic...
		if score > MaxScore {
			score = 0
		}
		// And the equally unlikely scenario of all hash reps yielding
		// all 0 bits would result in a score of 1 by the logic above.

		b.score = uint32(score)
		b.scored = true
	}
	return b.score
}

func onesCount256(value [32]byte) int {
	count := 0
	for _, byt := range value {
		count += bits.OnesCount8(byt)
	}
	return count
}

func (b *Block) Marshal() proto.Message {
	msg := &BlockMessage{
		Version:     b.GetVersion(),
		ID:          b.GetID(),
		ParentID:    b.GetParentID(),
		BlockNumber: b.GetBlockNumber(),
		Timestamp:   b.GetTimestamp(),
		PeerID:      b.GetPeerID(),
		Score:       b.GetScore()}

	msg.Transactions = make([]*any.Any, len(b.GetTransactions()))
	for i, t := range b.GetTransactions() {
		tMsg := t.Marshal()
		a, err := ptypes.MarshalAny(tMsg)
		if err != nil {
			//TOD
			return nil
		}

		msg.Transactions[i] = a
	}

	return msg
}

func (b *Block) Unmarshal(message proto.Message, txnHandlers map[string]spec.TransactionHandler) {
	msg := message.(*BlockMessage)

	b.id = msg.GetID()
	b.parentID = msg.GetParentID()
	b.blockNumber = msg.GetBlockNumber()
	b.timestamp = msg.GetTimestamp()
	b.peerID = msg.GetPeerID()
	b.score = msg.GetScore()

	txnMsgs := msg.GetTransactions()
	b.transactions = make([]spec.Transaction, len(txnMsgs))
	for i, any := range txnMsgs {
		txnType, err := ptypes.AnyMessageName(any)
		if err != nil {
			//TODO
			continue
		}
		h := txnHandlers[txnType]
		if h == nil {
			// TODO, how to handle
			continue
		}
		b.transactions[i] = h.Unmarshal(any)
	}
}

func (b *Block) GenerateID() {
	data := b.GetVersion() + b.GetPeerID() + b.GetParentID() +
		strconv.FormatUint(b.GetBlockNumber(), 10) +
		strconv.FormatInt(b.GetTimestamp(), 10) +
		strconv.FormatUint(uint64(b.GetScore()), 10)

	for _, t := range b.transactions {
		data += t.GetID()
	}

	hash := sha256.New()
	hash.Write([]byte(data))
	b.id = hex.EncodeToString(hash.Sum(nil))
}
