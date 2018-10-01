package luckyblock

import (
	"context"
	"sync"
	"time"

	spec "github.com/blocktop/go-spec"
	st "github.com/blocktop/go-store-ipfs"
)

type blockCommitter struct {
	sync.Mutex
	txnHandlers map[string]spec.TransactionHandler
}

func (c *blockCommitter) TryCommit(newBlock spec.Block, branch []spec.Block) (string, error) {
	c.Lock()
	defer c.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	store, err := st.Store.OpenBlock(newBlock.BlockNumber())
	if err != nil {
		return "", err
	}
	defer store.Revert()

	return c.addBranch(ctx, store, newBlock, branch)
}

func (c *blockCommitter) Commit(block spec.Block) (string, error) {
	c.Lock()
	defer c.Unlock()

	store, err := st.Store.OpenBlock(block.BlockNumber())
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	hash, err := c.addBlock(ctx, store, block)
	if err != nil {
		store.Revert()
		return "", err
	}
	err = store.Commit(ctx)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (c *blockCommitter) addBranch(ctx context.Context, store spec.StoreBlock, block spec.Block, branch []spec.Block) (string, error) {
	for i := len(branch) - 1; i >= 0; i-- {
		_, err := c.addBlock(ctx, store, branch[i])
		if err != nil {
			return "", err
		}
	}

	return c.addBlock(ctx, store, block)
}

func (c *blockCommitter) addBlock(ctx context.Context, store spec.StoreBlock, block spec.Block) (string, error) {
	return store.Submit(ctx, block)
}
