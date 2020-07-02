/**
* @file
* @copyright defined in scdo/LICENSE
 */

package svm

import (
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/contract/system"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/svm/evm"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/core/vm"
)

// Context for other vm constructs
type Context struct {
	Tx          *types.Transaction
	TxIndex     int
	Statedb     *state.Statedb
	BlockHeader *types.BlockHeader
	BcStore     store.BlockchainStore
}

// Process the tx
func Process(ctx *Context, height uint64) (*types.Receipt, error) {
	// check the tx against the latest statedb, e.g. balance, nonce.
	if err := ctx.Tx.ValidateState(ctx.Statedb, height); err != nil {
		return nil, errors.NewStackedError(err, "failed to validate tx against statedb")
	}

	// Pay intrinsic gas all the time
	gasLimit := ctx.Tx.Data.GasLimit
	intrGas := ctx.Tx.IntrinsicGas()
	if gasLimit < intrGas {
		return nil, types.ErrIntrinsicGas
	}
	leftOverGas := gasLimit - intrGas

	// init statedb and set snapshot
	var err error
	var receipt *types.Receipt
	snapshot := ctx.Statedb.Prepare(ctx.TxIndex)

	// create or execute contract
	if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil { // system contract
		receipt, err = processSystemContract(ctx, contract, snapshot, leftOverGas)
	} else if ctx.Tx.IsCrossShardTx() && !ctx.Tx.Data.To.IsEVMContract() { // cross shard tx
		return processCrossShardTransaction(ctx, snapshot)
	} else { // evm
		receipt, err = processEvmContract(ctx, leftOverGas, height)
	}
	// fmt.Println("svm.go-59, receipt.result", receipt.Result)
	// account balance is not enough (account.balance < tx.amount)
	if err == vm.ErrInsufficientBalance { // there is no effect to statedb, just revert to previous snapshot
		return nil, revertStatedb(ctx.Statedb, snapshot, err)
	}

	if err != nil {
		if height <= common.SmartContractNonceForkHeight {
			// fmt.Println("smart contract OLD logic")
			ctx.Statedb.RevertToSnapshot(snapshot)
			receipt.Failed = true
			receipt.Result = []byte(err.Error())

		} else {
			// fmt.Println("smart contract NEW logic")
			databaseAccountNonce := ctx.Statedb.GetNonce(ctx.Tx.Data.From)
			setNonce := databaseAccountNonce
			if ctx.Tx.Data.AccountNonce >= databaseAccountNonce {
				setNonce = ctx.Tx.Data.AccountNonce + 1
			}
			ctx.Statedb.RevertToSnapshot(snapshot)
			ctx.Statedb.SetNonce(ctx.Tx.Data.From, setNonce)
			receipt.Failed = true
			receipt.Result = []byte(err.Error())
		}

	}

	// include the intrinsic gas
	receipt.UsedGas += intrGas

	// refund gas, capped to half of the used gas.
	refund := ctx.Statedb.GetRefund()
	if maxRefund := receipt.UsedGas / 2; refund > maxRefund {
		refund = maxRefund
	}
	receipt.UsedGas -= refund

	return handleFee(ctx, receipt, snapshot)
}

func processCrossShardTransaction(ctx *Context, snapshot int) (*types.Receipt, error) {
	receipt := &types.Receipt{
		TxHash:  ctx.Tx.Hash,
		UsedGas: types.CrossShardTotalGas,
	}

	// Add from nonce
	ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)

	// Transfer amount
	amount, sender := ctx.Tx.Data.Amount, ctx.Tx.Data.From
	if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	ctx.Statedb.SubBalance(sender, amount)

	// check fee, only support non-contract tx.
	txFee := new(big.Int).Mul(ctx.Tx.Data.GasPrice, new(big.Int).SetUint64(receipt.UsedGas))
	if ctx.Statedb.GetBalance(sender).Cmp(txFee) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}
	receipt.TotalFee = txFee.Uint64()

	// handle fee
	ctx.Statedb.SubBalance(sender, txFee)
	minerFee := new(big.Int).Mul(ctx.Tx.Data.GasPrice, new(big.Int).SetUint64(types.CrossShardTransactionGas))
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, minerFee)

	// Record statedb hash
	var err error
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		err = errors.NewStackedError(err, "failed to get statedb root hash")
		return nil, revertStatedb(ctx.Statedb, snapshot, err)
	}
	return receipt, nil
}

func processSystemContract(ctx *Context, contract system.Contract, snapshot int, leftOverGas uint64) (*types.Receipt, error) {
	// must execute to make sure that system contract address is available
	if !ctx.Statedb.Exist(ctx.Tx.Data.To) {
		ctx.Statedb.CreateAccount(ctx.Tx.Data.To)
	}

	var err error
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	// Add from nonce
	ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)

	// Transfer amount
	amount, sender, recipient := ctx.Tx.Data.Amount, ctx.Tx.Data.From, ctx.Tx.Data.To
	if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 { //balance is not enough
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	ctx.Statedb.SubBalance(sender, amount)
	ctx.Statedb.AddBalance(recipient, amount)

	// Check used gas is over flow
	receipt.UsedGas = contract.RequiredGas(ctx.Tx.Data.Payload)
	if receipt.UsedGas > leftOverGas {
		return receipt, vm.ErrOutOfGas
	}
	// Run
	receipt.Result, err = contract.Run(ctx.Tx.Data.Payload, system.NewContext(ctx.Tx, ctx.Statedb, ctx.BlockHeader))

	return receipt, err
}

func processEvmContract(ctx *Context, gas uint64, height uint64) (*types.Receipt, error) {
	var err error
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	statedb := &evm.StateDB{Statedb: ctx.Statedb}
	e := evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)
	caller := vm.AccountRef(ctx.Tx.Data.From)
	var leftOverGas uint64

	// fmt.Println("ctx.Tx.Data.To.IsEmpty()?", ctx.Tx.Data.To.IsEmpty())
	if ctx.Tx.Data.To.IsEmpty() { // this is smart contract deployment
		var createdContractAddr common.Address
		receipt.Result, createdContractAddr, leftOverGas, err = e.Create(caller, ctx.Tx.Data.Payload, gas, ctx.Tx.Data.Amount)
		if !createdContractAddr.IsEmpty() {
			receipt.ContractAddress = createdContractAddr.Bytes()
		}
		// fmt.Println("processEvmContract.go-173: before correcting statedbNonce ", ctx.Statedb.GetNonce(ctx.Tx.Data.From))
		// Since in the e.Create function, the setNonce in a wrong way (but already set, when user setnonce will be wrong), correct way is to ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)
		// nonce := evm.StateDB.GetNonce(caller.Address())
		// evm.StateDB.SetNonce(caller.Address(), nonce+1)

		if height > common.SmartContractNonceFixHeight {
			if err == nil {
				dbnonce := ctx.Statedb.GetNonce(ctx.Tx.Data.From)
				// fmt.Println("before reset, statedb GetNonce", ctx.Statedb.GetNonce(ctx.Tx.Data.From))
				if dbnonce < ctx.Tx.Data.AccountNonce+1 { // here only need to compare dbnonce with accountnonce + 1, since dbnonce is already set
					ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1) // if before setting value is smaller than user setNonce value, then should reset.
					// fmt.Println("after, SetNonce to Tx.Data.AccountNonce + 1, namely", ctx.Tx.Data.AccountNonce+1)
				}
			}
		}

		// fmt.Println("processEvmContract.go-173:after correcting statedbNonce", ctx.Statedb.GetNonce(ctx.Tx.Data.From))
	} else {
		ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)
		// fmt.Printf("ToAddress is not empty, setNonce to %d", ctx.Tx.Data.AccountNonce+1)
		receipt.Result, leftOverGas, err = e.Call(caller, ctx.Tx.Data.To, ctx.Tx.Data.Payload, gas, ctx.Tx.Data.Amount)
	}
	receipt.UsedGas = gas - leftOverGas
	// fmt.Println("svm.go-183 processEVMContract [after Create] err: ", err)
	return receipt, err
}

func handleFee(ctx *Context, receipt *types.Receipt, snapshot int) (*types.Receipt, error) {
	// Calculating the total fee
	// For normal tx: fee = 20k * 1 Wen/gas = 0.0002 Scdo
	// For contract tx, average gas per tx is about 100k on ETH, fee = 100k * 1Wen/gas = 0.001 Scdo
	usedGas := new(big.Int).SetUint64(receipt.UsedGas)
	totalFee := new(big.Int).Mul(usedGas, ctx.Tx.Data.GasPrice)

	// Transfer fee to coinbase
	// Note, the sender should always have enough balance.
	ctx.Statedb.SubBalance(ctx.Tx.Data.From, totalFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, totalFee)
	receipt.TotalFee = totalFee.Uint64()

	// Record statedb hash
	var err error
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		err = errors.NewStackedError(err, "failed to get statedb root hash")
		return nil, revertStatedb(ctx.Statedb, snapshot, err)
	}

	// Add logs
	receipt.Logs = ctx.Statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}

	return receipt, nil
}

func revertStatedb(statedb *state.Statedb, snapshot int, err error) error {
	statedb.RevertToSnapshot(snapshot)
	return err
}
