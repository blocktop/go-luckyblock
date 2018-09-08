package luckyblock

import (
	"strconv"
	"crypto/sha256"
	"math/bits"


	spec "github.com/blckit/go-spec"
)

var BlockComparator spec.BlockComparator = func(blocks []spec.Block) spec.Block {

	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) == 1 {
		return blocks[0]
	}

	var comparator = func(b1, b2 spec.Block, round int) int {
		b1Luck := b1.(*Block)
		b2Luck := b2.(*Block)
		
		if round == 0 {
			if b1Luck.score > b2Luck.score {
				return 0
			}
			if b2Luck.score > b1Luck.score {
				return 1
			}
			return 2
		}

		// scores equal
		// result based on hash of peerID with opponent blockID
		b1Hash := sha256.Sum256([]byte(b1Luck.GetID() + strconv.FormatInt(int64(round), 10)))
		b2Hash := sha256.Sum256([]byte(b2Luck.GetID() + strconv.FormatInt(int64(round), 10)))
		b1Score := int(b1Luck.GetScore())
		b2Score := int(b2Luck.GetScore())
		for i := 0; i < 32; i++ {
			b1Score += bits.OnesCount8(b1Hash[i])
			b2Score += bits.OnesCount8(b2Hash[i])
		}

		if b1Score > b2Score {
			return 0
		}
		if b2Score > b1Score {
			return 1
		}
		return 2
	}

	// compare all blocks to each other
	result := make([]spec.Block, len(blocks))
	copy(result, blocks)

	round := 0
	for len(result) > 1 {
		i := 0
		j := i + 1
		for i < len(result)-1 {
			switch comparator(result[i], result[j], round) {
			case 0:
				result = append(result[:j], result[j+1:]...)
				// jth element is already the next block, so don't increment j
			case 1:
				result = append(result[:i], result[i+1:]...)
				j = i + 1 // reset j since ith element is now the next block
			case 2:
				j++
			}

			if j >= len(result)-1 {
				i++
				j = i + 1
			}
		}
		round++
	}

	return result[0]
}