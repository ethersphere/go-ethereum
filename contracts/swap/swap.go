// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

// Package swap wraps the 'swap' Ethereum smart contract.
// It is an abstraction layer to hide implementation details about the different
// Swap contract iterations (Simple Swap, Soft Swap, etc.)
package swap

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethersphere/swarm/contracts/swap/contract"
)

var (
	// ErrNotASwapContract is given when an address is verified not to have a SWAP contract based on its bytecode
	ErrNotASwapContract = errors.New("not a swap contract")
	// ErrTransactionReverted is given when the transaction that submits or cashes a cheque is reverted
	ErrTransactionReverted = errors.New("Transaction reverted")
)

// Validator struct -> put validator in implementation of Swap. Make the validator a package level function and implement this in Swap

// Backend wraps all methods required for contract deployment.
type Backend interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	//TODO: needed? BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error)
}

// SimpleSwap interface defines the simple swap's exposed methods
type SimpleSwap interface {
	Deploy(auth *bind.TransactOpts, backend bind.ContractBackend, owner common.Address, harddepositTimeout *big.Int) (common.Address, *types.Transaction, error)
	SubmitChequeBeneficiary(opts *bind.TransactOpts, serial *big.Int, amount *big.Int, timeout *big.Int, ownerSig []byte) (*types.Transaction, error)
	CashChequeBeneficiary(auth *bind.TransactOpts, backend Backend, beneficiary common.Address, requestPayout *big.Int) (*types.Transaction, error)
	ValidateCode() bool
	ContractParams() *Params
	InstanceAt(address common.Address, backend bind.ContractBackend)
}

// Swap is a proxy object for Swap contracts.
type Swap struct {
	Instance *contract.SimpleSwap
}

// Params encapsulates some contract parameters (currently mostly informational)
type Params struct {
	ContractCode, ContractAbi string
}

// New returns a pointer to a new Swap struct
func New() *Swap {
	return &Swap{}
}

// ValidateCode checks that the on-chain code at address matches the expected swap
// contract code.
// TODO: have this as a package level function and pass the SimpleSwapBin as argument
func (s *Swap) ValidateCode(ctx context.Context, b bind.ContractBackend, address common.Address) error {
	codeReadFromAddress, err := b.CodeAt(ctx, address, nil)
	if err != nil {
		return err
	}
	referenceCode := common.FromHex(contract.ContractDeployedCode)
	if !bytes.Equal(codeReadFromAddress, referenceCode) {
		return ErrNotASwapContract
	}
	return nil
}

// Deploy a Swap contract
func Deploy(auth *bind.TransactOpts, backend bind.ContractBackend, owner common.Address, harddepositTimeout time.Duration) (addr common.Address, s *Swap, tx *types.Transaction, err error) {
	s = New()
	addr, tx, s.Instance, err = contract.DeploySimpleSwap(auth, backend, owner, big.NewInt(int64(harddepositTimeout.Seconds())))
	return addr, s, tx, err
}

// ContractParams returns contract information
func (s *Swap) ContractParams() *Params {
	return &Params{
		ContractCode: contract.SimpleSwapBin,
		ContractAbi:  contract.SimpleSwapABI,
	}
}

// waitForTx waits for transaction to be mined and returns the receipt
func waitForTx(auth *bind.TransactOpts, backend Backend, tx *types.Transaction) (*types.Receipt, error) {
	// it blocks here until tx is mined
	receipt, err := bind.WaitMined(auth.Context, backend, tx)
	if err != nil {
		return nil, err
	}
	// indicate whether the transaction did not revert
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, ErrTransactionReverted
	}
	return receipt, nil
}

// SubmitChequeBeneficiary prepares to send a call to submitChequeBeneficiary and blocks until the transaction is mined.
func (s *Swap) SubmitChequeBeneficiary(auth *bind.TransactOpts, backend Backend, serial *big.Int, amount *big.Int, timeout *big.Int, ownerSig []byte) (*types.Receipt, error) {
	tx, err := s.Instance.SubmitChequeBeneficiary(auth, serial, amount, timeout, ownerSig)
	if err != nil {
		return nil, err
	}
	return waitForTx(auth, backend, tx)
}

// CashChequeBeneficiary cashes the cheque.
func (s *Swap) CashChequeBeneficiary(auth *bind.TransactOpts, backend Backend, beneficiary common.Address, requestPayout *big.Int) (*types.Receipt, error) {
	tx, err := s.Instance.CashChequeBeneficiary(auth, beneficiary, requestPayout)
	if err != nil {
		return nil, err
	}
	return waitForTx(auth, backend, tx)
}

// InstanceAt returns a new instance of simpleSwap at the address which was given
func InstanceAt(address common.Address, backend bind.ContractBackend) (s *Swap, err error) {
	s = New()
	s.Instance, err = contract.NewSimpleSwap(address, backend)
	return s, err
}
