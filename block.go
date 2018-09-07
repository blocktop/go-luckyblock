package luckyblock

import (
	"strconv"
	"crypto/sha256"
	"encoding/hex"
	"math/bits"
	"time"

	spec "github.com/blckit/go-spec"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

const (
	BlockType    = "luckyblock"
	BlockVersion = "v1"
)

type Block struct {
	id, parentID string
	blockNumber  uint64
	score        uint32
	transactions []spec.Transaction
	timestamp    int64
	peerID       string
}

func NewBlock(parent *Block, peerID string) *Block {
	b := &Block{}
	b.peerID = peerID
	b.timestamp = time.Now().UnixNano() / int64(time.Millisecond)

	if (parent == nil) {
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
	return BlockType
}

func (b *Block) GetVersion() string {
	return BlockVersion
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
	if b.score == 0 {
		hash := sha256.Sum256([]byte(b.peerID + b.parentID))
		b.score = 0
		for _, byt := range hash {
			b.score += uint32(bits.OnesCount8(byt))
		}
	}
	return b.score
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
		b.transactions[i] = h.UnmarshalAny(any)
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
