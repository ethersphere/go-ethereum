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
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	contract "github.com/ethersphere/go-sw3/contracts-v0-1-1/simpleswap"
)

var (
	// ErrTransactionReverted is given when the transaction that cashes a cheque is reverted
	ErrTransactionReverted = errors.New("Transaction reverted")
)

// Backend wraps all methods required for contract deployment.
type Backend interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// Contract interface defines the methods exported from the underlying go-bindings for the smart contract
type Contract interface {
	// Withdraw attempts to withdraw Wei from the chequebook
	Withdraw(auth *bind.TransactOpts, backend Backend, amount *big.Int) (*types.Receipt, error)
	// Deposit sends a raw transaction to the chequebook, triggering the fallback—depositing amount
	Deposit(auth *bind.TransactOpts, backend Backend, amout *big.Int) (*types.Receipt, error)
	// CashChequeBeneficiary cashes the cheque by the beneficiary
	CashChequeBeneficiary(auth *bind.TransactOpts, beneficiary common.Address, cumulativePayout *big.Int, ownerSig []byte) (*CashChequeResult, *types.Receipt, error)
	// LiquidBalance returns the LiquidBalance (total balance in Wei - total hard deposits in Wei) of the chequebook
	LiquidBalance(auth *bind.CallOpts) (*big.Int, error)
	// ContractParams returns contract info (e.g. deployed address)
	ContractParams() *Params
	// Issuer returns the contract owner from the blockchain
	Issuer(opts *bind.CallOpts) (common.Address, error)
	// PaidOut returns the total paid out amount for the given address
	PaidOut(opts *bind.CallOpts, addr common.Address) (*big.Int, error)
}

// CashChequeResult summarizes the result of a CashCheque or CashChequeBeneficiary call
type CashChequeResult struct {
	Beneficiary      common.Address // beneficiary of the cheque
	Recipient        common.Address // address which received the funds
	Caller           common.Address // caller of cashCheque
	TotalPayout      *big.Int       // total amount that was paid out in this call
	CumulativePayout *big.Int       // cumulative payout of the cheque that was cashed
	CallerPayout     *big.Int       // payout for the caller of cashCheque
	Bounced          bool           // indicates wether parts of the cheque bounced
}

// Params encapsulates some contract parameters (currently mostly informational)
type Params struct {
	ContractCode    string
	ContractAbi     string
	ContractAddress common.Address
}

type simpleContract struct {
	instance *contract.SimpleSwap
	address  common.Address
	backend  Backend
}

// Deploy deploys an instance of the underlying contract and returns its instance and the transaction identifier
func Deploy(auth *bind.TransactOpts, backend Backend, owner common.Address, harddepositTimeout time.Duration) (Contract, *types.Transaction, error) {
	addr, tx, instance, err := contract.DeploySimpleSwap(auth, backend, owner, big.NewInt(int64(harddepositTimeout)))
	c := simpleContract{instance: instance, address: addr, backend: backend}
	return c, tx, err
}

// InstanceAt creates a new instance of a contract at a specific address.
// It assumes that there is an existing contract instance at the given address, or an error is returned
// This function is needed to communicate with remote Swap contracts (e.g. sending a cheque)
func InstanceAt(address common.Address, backend Backend) (Contract, error) {
	instance, err := contract.NewSimpleSwap(address, backend)
	if err != nil {
		return nil, err
	}
	c := simpleContract{instance: instance, address: address, backend: backend}
	return c, err
}

// Withdraw withdraws amount from the chequebook and blocks until the transaction is mined
func (s simpleContract) Withdraw(auth *bind.TransactOpts, backend Backend, amount *big.Int) (*types.Receipt, error) {
	tx, err := s.instance.Withdraw(auth, amount)
	if err != nil {
		return nil, err
	}
	return WaitFunc(auth, backend, tx)
}

// Deposit sends a transaction to the chequebook, which deposits the amount set in Auth.Value and blocks until the transaction is mined
func (s simpleContract) Deposit(auth *bind.TransactOpts, backend Backend, amount *big.Int) (*types.Receipt, error) {
	rawSimpleSwap := contract.SimpleSwapRaw{Contract: s.instance}
	if auth.Value != big.NewInt(0) {
		return nil, fmt.Errorf("Deposit value can only be set via amount parameter")
	}
	if amount == big.NewInt(0) {
		return nil, fmt.Errorf("Deposit amount cannot be equal to zero")
	}
	auth.Value = amount
	tx, err := rawSimpleSwap.Transfer(auth)
	if err != nil {
		return nil, err
	}
	return WaitFunc(auth, backend, tx)
}

// CashChequeBeneficiary cashes the cheque on the blockchain and blocks until the transaction is mined.
func (s simpleContract) CashChequeBeneficiary(opts *bind.TransactOpts, beneficiary common.Address, cumulativePayout *big.Int, ownerSig []byte) (*CashChequeResult, *types.Receipt, error) {
	tx, err := s.instance.CashChequeBeneficiary(opts, beneficiary, cumulativePayout, ownerSig)
	if err != nil {
		return nil, nil, err
	}
	receipt, err := WaitFunc(opts, s.backend, tx)
	if err != nil {
		return nil, nil, err
	}

	result := &CashChequeResult{
		Bounced: false,
	}

	for _, log := range receipt.Logs {
		if log.Address != s.address {
			continue
		}
		if event, err := s.instance.ParseChequeCashed(*log); err == nil {
			result.Beneficiary = event.Beneficiary
			result.Caller = event.Caller
			result.CallerPayout = event.CallerPayout
			result.TotalPayout = event.TotalPayout
			result.CumulativePayout = event.CumulativePayout
			result.Recipient = event.Recipient
		} else if _, err := s.instance.ParseChequeBounced(*log); err == nil {
			result.Bounced = true
		}
	}

	return result, receipt, nil
}

// LiquidBalance returns the LiquidBalance (total balance in Wei - total hard deposits in Wei) of the chequebook
func (s simpleContract) LiquidBalance(opts *bind.CallOpts) (*big.Int, error) {
	return s.instance.LiquidBalance(opts)
}

// ContractParams returns contract information
func (s simpleContract) ContractParams() *Params {
	return &Params{
		ContractCode:    contract.SimpleSwapBin,
		ContractAbi:     contract.SimpleSwapABI,
		ContractAddress: s.address,
	}
}

// Issuer returns the contract owner from the blockchain
func (s simpleContract) Issuer(opts *bind.CallOpts) (common.Address, error) {
	return s.instance.Issuer(opts)
}

// PaidOut returns the total paid out amount for the given address
func (s simpleContract) PaidOut(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	return s.instance.PaidOut(opts, addr)
}

// WaitFunc is the default function to wait for transactions
// We can overwrite this in tests so that we don't need to wait for mining
var WaitFunc = waitForTx

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
