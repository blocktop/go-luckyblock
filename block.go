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
	"fmt"
	"math/bits"
	"strconv"
	"time"

	"github.com/spf13/viper"

	spec "github.com/blocktop/go-spec"
	"github.com/golang/protobuf/proto"
)

const (
	MaxScore = uint64(^uint32(0))
)

type Block struct {
	id, parentID string
	blockNumber  uint64
	score        uint32
	scored       bool
	txns         []spec.Transaction
	txnHashes    []string
	timestamp    int64
	peerID       string
	name         string
	namespace    string
	blockVersion string
}

var _ spec.Block = (*Block)(nil)

func NewBlock(parent *Block, peerID string) *Block {
	b := &Block{}
	b.peerID = peerID
	b.timestamp = time.Now().UnixNano() / int64(time.Millisecond)
	b.name = viper.GetString("blockchain.block.name")
	b.namespace = viper.GetString("blockchain.block.namespace")
	b.blockVersion = viper.GetString("blockchain.block.version")

	if parent == nil {
		// genesis block or prototype
		b.blockNumber = uint64(0)
		b.parentID = ""
	} else {
		b.blockNumber = parent.BlockNumber() + uint64(1)
		b.parentID = parent.Hash()
	}

	return b
}

func (b *Block) ResourceType() string {
	return "block"
}

func (b *Block) Namespace() string {
	return b.namespace
}

func (b *Block) Name() string {
	return b.name
}

func (b *Block) Version() string {
	return b.blockVersion
}

func (b *Block) Hash() string {
	if b.id == "" {
		data, links, err := b.Marshal()
		h := sha256.New()
		_, err = h.Write(data)
		if err != nil {
			return ""
		}

		for _, linkHash := range links {
			h.Write([]byte(linkHash))
		}

		b.id = hex.EncodeToString(h.Sum(nil))
	}
	return b.id
}

func (b *Block) ParentHash() string {
	return b.parentID
}

func (b *Block) BlockNumber() uint64 {
	return b.blockNumber
}

func (b *Block) Valid() bool {
	return true
}

func (b *Block) Transactions() []spec.Transaction {
	if b.txns == nil {
		//TODO build txns from hashes
		b.txns = make([]spec.Transaction, 0)
	}
	return b.txns
}

func (b *Block) Timestamp() int64 {
	return b.timestamp
}

func (b *Block) PeerID() string {
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

func (b *Block) Marshal() ([]byte, spec.Links, error) {
	msg := &BlockMessage{
		Name:        b.Name(),
		Namespace:   b.Namespace(),
		Version:     b.Version(),
		BlockNumber: b.BlockNumber(),
		Timestamp:   b.Timestamp(),
		PeerID:      b.PeerID(),
		Score:       b.GetScore()}

	links := make(map[string]string)
	links["parent"] = b.ParentHash()

	for i, t := range b.Transactions() {
		key := fmt.Sprintf("txn-%d", i+1)
		links[key] = t.Hash()
	}

	msg.Links = links

	byts, err := proto.Marshal(msg)
	if err != nil {
		return nil, nil, err
	}

	return byts, links, nil
}

func (b *Block) Unmarshal(data []byte, links spec.Links) error {
	msg := &BlockMessage{}
	err := proto.Unmarshal(data, msg)
	if err != nil {
		return err
	}

	b.name = msg.GetName()
	b.namespace = msg.GetNamespace()
	b.blockVersion = msg.GetVersion()
	b.blockNumber = msg.GetBlockNumber()
	b.timestamp = msg.GetTimestamp()
	b.peerID = msg.GetPeerID()
	b.score = msg.GetScore()
	b.scored = true

	b.parentID = links["parent"]
	b.txns = nil
	b.txnHashes = make([]string, len(links)-1)
	i := 0
	for k, tx := range links {
		if k != "parent" {
			b.txnHashes[i] = tx
			i++
		}
	}

	return nil
}
