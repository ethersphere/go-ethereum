// Copyright 2020 The Swarm Authors
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

package swap

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	contract "github.com/ethersphere/swarm/contracts/swap"
)

func newSignedTestCheque(testChequeContract common.Address, beneficiaryAddress common.Address, cumulativePayout *big.Int, signingKey *ecdsa.PrivateKey) (*Cheque, error) {
	cheque := &Cheque{
		ChequeParams: ChequeParams{
			Contract:         testChequeContract,
			CumulativePayout: cumulativePayout.Uint64(),
			Beneficiary:      beneficiaryAddress,
		},
		Honey: cumulativePayout.Uint64(),
	}

	sig, err := cheque.Sign(signingKey)
	if err != nil {
		return nil, err
	}
	cheque.Signature = sig
	return cheque, nil
}

// TestContractIntegration tests a end-to-end cheque interaction.
// First a simulated backend is created, then we deploy the issuer's swap contract.
// We issue a test cheque with the beneficiary address and on the issuer's contract,
// and immediately try to cash-in the cheque
// afterwards it attempts to cash-in a bouncing cheque
func TestContractIntegration(t *testing.T) {
	backend := newTestBackend(t)
	reset := setupContractTest()
	defer reset()

	payout := big.NewInt(42)

	chequebook, err := testDeployWithPrivateKey(context.Background(), backend, ownerKey, ownerAddress, payout)
	if err != nil {
		t.Fatal(err)
	}

	cheque, err := newSignedTestCheque(chequebook.ContractParams().ContractAddress, beneficiaryAddress, payout, ownerKey)
	if err != nil {
		t.Fatal(err)
	}

	opts := bind.NewKeyedTransactor(beneficiaryKey)

	tx, err := chequebook.CashChequeBeneficiaryStart(opts, beneficiaryAddress, payout, cheque.Signature)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err := contract.WaitForTransactionByHash(context.Background(), backend, tx.Hash())
	if err != nil {
		t.Fatal(err)
	}

	cashResult := chequebook.CashChequeBeneficiaryResult(receipt)
	if receipt.Status != 1 {
		t.Fatalf("Bad status %d", receipt.Status)
	}
	if cashResult.Bounced {
		t.Fatal("cashing bounced")
	}

	// check state, check that cheque is indeed there
	result, err := chequebook.PaidOut(nil, beneficiaryAddress)
	if err != nil {
		t.Fatal(err)
	}
	if result.Uint64() != cheque.CumulativePayout {
		t.Fatalf("Wrong cumulative payout %d", result)
	}
	log.Debug("cheques result", "result", result)

	// create a cheque that will bounce
	bouncingCheque, err := newSignedTestCheque(chequebook.ContractParams().ContractAddress, beneficiaryAddress, payout.Add(payout, big.NewInt(int64(10000*RetrieveRequestPrice))), ownerKey)
	if err != nil {
		t.Fatal(err)
	}

	tx, err = chequebook.CashChequeBeneficiaryStart(opts, beneficiaryAddress, big.NewInt(int64(bouncingCheque.CumulativePayout)), bouncingCheque.Signature)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err = contract.WaitForTransactionByHash(context.Background(), backend, tx.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 {
		t.Fatalf("Bad status %d", receipt.Status)
	}

	cashResult = chequebook.CashChequeBeneficiaryResult(receipt)
	if !cashResult.Bounced {
		t.Fatal("cheque did not bounce")
	}

}

func TestCashCheque(t *testing.T) {
	backend := newTestBackend(t)
	reset := setupContractTest()
	defer reset()

	cashoutProcessor := newCashoutProcessor(backend, ownerKey)
	payout := big.NewInt(42)

	chequebook, err := testDeployWithPrivateKey(context.Background(), backend, ownerKey, ownerAddress, payout)
	if err != nil {
		t.Fatal(err)
	}

	testCheque, err := newSignedTestCheque(chequebook.ContractParams().ContractAddress, ownerAddress, payout, ownerKey)
	if err != nil {
		t.Fatal(err)
	}

	err = cashoutProcessor.cashCheque(context.Background(), &CashoutRequest{
		Cheque:      *testCheque,
		Destination: ownerAddress,
	})
	if err != nil {
		t.Fatal(err)
	}

	paidOut, err := chequebook.PaidOut(nil, ownerAddress)
	if err != nil {
		t.Fatal(err)
	}

	if paidOut.Cmp(big.NewInt(int64(testCheque.CumulativePayout))) != 0 {
		t.Fatalf("paidOut does not equal the CumulativePayout: paidOut=%v expected=%v", paidOut, testCheque.CumulativePayout)
	}
}

func TestEstimatePayout(t *testing.T) {
	backend := newTestBackend(t)
	reset := setupContractTest()
	defer reset()

	cashoutProcessor := newCashoutProcessor(backend, ownerKey)
	payout := big.NewInt(42)

	chequebook, err := testDeployWithPrivateKey(context.Background(), backend, ownerKey, ownerAddress, payout)
	if err != nil {
		t.Fatal(err)
	}

	testCheque, err := newSignedTestCheque(chequebook.ContractParams().ContractAddress, ownerAddress, payout, ownerKey)
	if err != nil {
		t.Fatal(err)
	}

	expectedPayout, transactionCost, err := cashoutProcessor.estimatePayout(context.Background(), testCheque)
	if err != nil {
		t.Fatal(err)
	}

	if expectedPayout != payout.Uint64() {
		t.Fatalf("unexpected expectedPayout: got %d, wanted: %d", expectedPayout, payout.Uint64())
	}

	// the gas price in the simulated backend is 1 therefore the total transactionCost should be 50000 * 1 = 50000
	if transactionCost != 50000 {
		t.Fatalf("unexpected transactionCost: got %d, wanted: %d", transactionCost, 0)
	}
}
