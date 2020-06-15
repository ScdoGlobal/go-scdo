package seeleCredo

import (
	"math/big"

	"github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/seeleCredo/download"
)

type SlcBackend struct {
	s *ScdoService
}

// NewScdoBackend backend
func NewScdoBackend(s *ScdoService) *SlcBackend {
	return &SlcBackend{s}
}

// TxPoolBackend tx pool
func (sd *SlcBackend) TxPoolBackend() api.Pool { return sd.s.txPool }

// GetNetVersion net version
func (sd *SlcBackend) GetNetVersion() string { return sd.s.netVersion }

// GetNetWorkID net id
func (sd *SlcBackend) GetNetWorkID() string { return sd.s.networkID }

// GetP2pServer p2p server
func (sd *SlcBackend) GetP2pServer() *p2p.Server { return sd.s.p2pServer }

// ChainBackend block chain db
func (sd *SlcBackend) ChainBackend() api.Chain { return sd.s.chain }

// Log return log pointer
func (sd *SlcBackend) Log() *log.ScdoLog { return sd.s.log }

// IsSyncing check status
func (sd *SlcBackend) IsSyncing() bool {
	seeleserviceAPI := sd.s.APIs()[5]
	d := seeleserviceAPI.Service.(downloader.PrivatedownloaderAPI)

	return d.IsSyncing()
}

// ProtocolBackend return protocol
func (sd *SlcBackend) ProtocolBackend() api.Protocol { return sd.s.seeleProtocol }

// GetBlock returns the requested block by hash or height
func (sd *SlcBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	var block *types.Block
	var err error
	if !hash.IsEmpty() {
		store := sd.s.chain.GetStore()
		block, err = store.GetBlock(hash)
		if err != nil {
			return nil, err
		}
	} else {
		if height < 0 {
			header := sd.s.chain.CurrentHeader()
			block, err = sd.s.chain.GetStore().GetBlockByHeight(header.Height)
		} else {
			block, err = sd.s.chain.GetStore().GetBlockByHeight(uint64(height))
		}
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

// GetBlockTotalDifficulty return total difficulty
func (sd *SlcBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	store := sd.s.chain.GetStore()
	return store.GetBlockTotalDifficulty(hash)
}

// GetReceiptByTxHash get receipt by transaction hash
func (sd *SlcBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	store := sd.s.chain.GetStore()
	receipt, err := store.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// GetTransaction return tx
func (sd *SlcBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, error) {
	return api.GetTransaction(pool, bcStore, txHash)
}
