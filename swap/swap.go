// Copyright 2018 The Swarm Authors
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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethersphere/swarm/contracts/swap"
	contract "github.com/ethersphere/swarm/contracts/swap"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/p2p/protocols"
	"github.com/ethersphere/swarm/state"
)

// ErrInvalidChequeSignature indicates the signature on the cheque was invalid
var ErrInvalidChequeSignature = errors.New("invalid cheque signature")

// Swap represents the Swarm Accounting Protocol
// a peer to peer micropayment system
// A node maintains an individual balance with every peer
// Only messages which have a price will be accounted for
type Swap struct {
	api                 PublicAPI
	store               state.Store          // store is needed in order to keep balances and cheques across sessions
	lock                sync.RWMutex         // lock the store
	balances            map[enode.ID]int64   // map of balances for each peer
	cheques             map[enode.ID]*Cheque // map of cheques for each peer
	peers               map[enode.ID]*Peer   // map of all swap Peers
	backend             contract.Backend     // the backend (blockchain) used
	owner               *Owner               // contract access
	params              *Params              // economic and operational parameters
	contract            swap.Contract        // reference to the smart contract
	oracle              PriceOracle          // the oracle providing the ether price for honey
	paymentThreshold    int64                // balance difference required for sending cheque
	disconnectThreshold int64                // balance difference required for dropping peer
}

// Owner encapsulates information related to accessing the contract
type Owner struct {
	Contract   common.Address    // address of swap contract
	address    common.Address    // owner address
	privateKey *ecdsa.PrivateKey // private key
	publicKey  *ecdsa.PublicKey  // public key
}

// Params encapsulates param
type Params struct {
	InitialDepositAmount uint64 //
}

// NewParams returns a Params struct filled with default values
func NewParams() *Params {
	return &Params{
		InitialDepositAmount: DefaultInitialDepositAmount,
	}
}

// New - swap constructor
func New(stateStore state.Store, prvkey *ecdsa.PrivateKey, contract common.Address, backend contract.Backend) *Swap {
	sw := &Swap{
		store:               stateStore,
		balances:            make(map[enode.ID]int64),
		backend:             backend,
		cheques:             make(map[enode.ID]*Cheque),
		peers:               make(map[enode.ID]*Peer),
		params:              NewParams(),
		paymentThreshold:    DefaultPaymentThreshold,
		disconnectThreshold: DefaultDisconnectThreshold,
		oracle:              NewPriceOracle(),
	}
	sw.owner = sw.createOwner(prvkey, contract)
	return sw
}

const (
	balancePrefix        = "balance_"
	sentChequePrefix     = "sent_cheque_"
	receivedChequePrefix = "received_cheque_"
)

// returns the store key for retrieving a peer's balance
func balanceKey(peer enode.ID) string {
	return balancePrefix + peer.String()
}

// returns the store key for retrieving a peer's last sent cheque
func sentChequeKey(peer enode.ID) string {
	return sentChequePrefix + peer.String()
}

// returns the store key for retrieving a peer's last received cheque
func receivedChequeKey(peer enode.ID) string {
	return receivedChequePrefix + peer.String()
}

func keyToID(key string, prefix string) enode.ID {
	return enode.HexID(key[len(prefix):])
}

// createOwner assings keys and addresses
func (s *Swap) createOwner(prvkey *ecdsa.PrivateKey, contract common.Address) *Owner {
	pubkey := &prvkey.PublicKey
	return &Owner{
		privateKey: prvkey,
		publicKey:  pubkey,
		Contract:   contract,
		address:    crypto.PubkeyToAddress(*pubkey),
	}
}

// DeploySuccess is for convenience log output
func (s *Swap) DeploySuccess() string {
	return fmt.Sprintf("contract: %s, owner: %s, deposit: %v, signer: %x", s.owner.Contract.Hex(), s.owner.address.Hex(), s.params.InitialDepositAmount, s.owner.publicKey)
}

// Add is the (sole) accounting function
// Swap implements the protocols.Balance interface
func (s *Swap) Add(amount int64, peer *protocols.Peer) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// load existing balances from the state store
	err = s.loadBalance(peer.ID())
	if err != nil && err != state.ErrNotFound {
		return fmt.Errorf("error while loading balance for peer %s", peer.ID().String())
	}

	// Check if balance with peer is over the disconnect threshold
	// It is the creditor who triggers the disconnect from a overdraft creditor,
	// thus we check for a positive value
	if s.balances[peer.ID()] >= s.disconnectThreshold {
		// if so, return error in order to abort the transfer
		return fmt.Errorf("balance for peer %s is over the disconnect threshold %d, disconnecting", peer.ID().String(), s.disconnectThreshold)
	}

	// calculate new balance
	var newBalance int64
	newBalance, err = s.updateBalance(peer.ID(), amount)
	if err != nil {
		return err
	}

	// Check if balance with peer crosses the threshold
	// It is the peer with a negative balance who sends a cheque, thus we check
	// that the balance is *below* the threshold
	if newBalance <= -s.paymentThreshold {
		//if so, send cheque
		log.Warn("balance for peer went over the payment threshold, sending cheque", "peer", peer.ID().String(), "payment threshold", s.paymentThreshold)
		return s.sendCheque(peer.ID())
	}

	return nil
}

// handleMsg is for handling messages when receiving messages
func (s *Swap) handleMsg(p *Peer) func(ctx context.Context, msg interface{}) error {
	return func(ctx context.Context, msg interface{}) error {
		switch msg := msg.(type) {
		case *EmitChequeMsg:
			go s.handleEmitChequeMsg(ctx, p, msg)
		}
		return nil
	}
}

// handleEmitChequeMsg should be handled by the creditor when it receives
// a cheque from a debitor
func (s *Swap) handleEmitChequeMsg(ctx context.Context, p *Peer, msg *EmitChequeMsg) error {
	cheque := msg.Cheque
	log.Info("received cheque from peer", "peer", p.ID().String())
	actualAmount, err := s.processAndVerifyCheque(cheque, p)
	if err != nil {
		return err
	}

	log.Debug("received cheque processed and verified", "peer", p.ID().String())

	// reset balance by amount
	// as this is done by the creditor, receiving the cheque, the amount should be negative,
	// so that updateBalance will calculate balance + amount which result in reducing the peer's balance
	s.lock.Lock()
	err = s.resetBalance(p.ID(), 0-int64(cheque.Honey))
	s.lock.Unlock()
	if err != nil {
		return err
	}

	// cash in cheque
	opts := bind.NewKeyedTransactor(s.owner.privateKey)
	opts.Context = ctx

	otherSwap, err := contract.InstanceAt(cheque.Contract, s.backend)
	if err != nil {
		return err
	}

	// submit cheque to the blockchain and cashes it directly
	go func() {
		// blocks here, as we are waiting for the transaction to be mined
		receipt, err := otherSwap.SubmitChequeBeneficiary(opts, s.backend, big.NewInt(int64(cheque.Serial)), big.NewInt(int64(cheque.Amount)), big.NewInt(int64(cheque.Timeout)), cheque.Signature)
		if err != nil {
			// TODO: do something with the error
			// and we actually need to log this error as we are in an async routine; nobody is handling this error for now
			log.Error("error submitting cheque", "err", err)
			return
		}
		log.Debug("submit tx mined", "receipt", receipt)

		receipt, err = otherSwap.CashChequeBeneficiary(opts, s.backend, s.owner.Contract, big.NewInt(int64(actualAmount)))
		if err != nil {
			// TODO: do something with the error
			// and we actually need to log this error as we are in an async routine; nobody is handling this error for now
			log.Error("error cashing cheque", "err", err)
			return
		}
		log.Info("Cheque successfully submitted and cashed")
	}()
	return err
}

// processAndVerifyCheque verifies the cheque and compares it with the last received cheque
// if the cheque is valid it will also be saved as the new last cheque
func (s *Swap) processAndVerifyCheque(cheque *Cheque, p *Peer) (uint64, error) {
	if err := s.verifyChequeProperties(cheque, p); err != nil {
		return 0, err
	}

	lastCheque := s.loadLastReceivedCheque(p)

	// TODO: there should probably be a lock here?
	expectedAmount, err := s.oracle.GetPrice(cheque.Honey)
	if err != nil {
		return 0, err
	}

	actualAmount, err := verifyChequeAgainstLast(cheque, lastCheque, expectedAmount)
	if err != nil {
		return 0, err
	}

	if err := s.saveLastReceivedCheque(p, cheque); err != nil {
		log.Error("error while saving last received cheque", "peer", p.ID().String(), "err", err.Error())
		// TODO: what do we do here? Related issue: https://github.com/ethersphere/swarm/issues/1515
	}

	return actualAmount, nil
}

// verifyChequeProperties verifies the signature and if the cheque fields are appropriate for this peer
// it does not verify anything that requires knowing the previous cheque
func (s *Swap) verifyChequeProperties(cheque *Cheque, p *Peer) error {
	if cheque.Contract != p.contractAddress {
		return fmt.Errorf("wrong cheque parameters: expected contract: %x, was: %x", p.contractAddress, cheque.Contract)
	}

	// the beneficiary is the owner of the counterparty swap contract
	if err := cheque.VerifySig(p.beneficiary); err != nil {
		return err
	}

	if cheque.Beneficiary != s.owner.address {
		return fmt.Errorf("wrong cheque parameters: expected beneficiary: %x, was: %x", s.owner.address, cheque.Beneficiary)
	}

	if cheque.Timeout != 0 {
		return fmt.Errorf("wrong cheque parameters: expected timeout to be 0, was: %d", cheque.Timeout)
	}

	return nil
}

// verifyChequeAgainstLast verifies that serial and amount are higher than in the previous cheque
// furthermore it cheques that the increase in amount is as expected
// returns the actual amount received in this cheque
func verifyChequeAgainstLast(cheque *Cheque, lastCheque *Cheque, expectedAmount uint64) (uint64, error) {
	actualAmount := cheque.Amount

	if lastCheque != nil {
		if cheque.Serial <= lastCheque.Serial {
			return 0, fmt.Errorf("wrong cheque parameters: expected serial larger than %d, was: %d", lastCheque.Serial, cheque.Serial)
		}

		if cheque.Amount <= lastCheque.Amount {
			return 0, fmt.Errorf("wrong cheque parameters: expected amount larger than %d, was: %d", lastCheque.Amount, cheque.Amount)
		}

		actualAmount -= lastCheque.Amount
	}

	if expectedAmount != actualAmount {
		return 0, fmt.Errorf("unexpected amount for honey, expected %d was %d", expectedAmount, actualAmount)
	}

	return actualAmount, nil
}

func (s *Swap) updateBalance(peer enode.ID, amount int64) (int64, error) {
	//adjust the balance
	//if amount is negative, it will decrease, otherwise increase
	s.balances[peer] += amount
	//save the new balance to the state store
	peerBalance := s.balances[peer]
	err := s.store.Put(balanceKey(peer), &peerBalance)
	if err != nil {
		return 0, fmt.Errorf("error while storing balance for peer %s", peer.String())
	}
	log.Debug("balance for peer after accounting", "peer", peer.String(), "balance", strconv.FormatInt(peerBalance, 10))
	return peerBalance, err
}

// loadBalance loads balances from the state store (persisted)
func (s *Swap) loadBalance(peer enode.ID) (err error) {
	var peerBalance int64
	if _, ok := s.balances[peer]; !ok {
		err = s.store.Get(balanceKey(peer), &peerBalance)
		s.balances[peer] = peerBalance
	}
	return
}

// sendCheque sends a cheque to peer
func (s *Swap) sendCheque(peer enode.ID) error {
	swapPeer, ok := s.getPeer(peer)
	if !ok {
		return fmt.Errorf("error while getting peer: %s", peer)
	}
	cheque, err := s.createCheque(peer)
	if err != nil {
		return fmt.Errorf("error while creating cheque: %s", err.Error())
	}

	log.Info("sending cheque", "serial", cheque.ChequeParams.Serial, "amount", cheque.ChequeParams.Amount, "beneficiary", cheque.Beneficiary, "contract", cheque.Contract)
	s.cheques[peer] = cheque

	err = s.store.Put(sentChequeKey(peer), &cheque)
	if err != nil {
		return fmt.Errorf("error while storing the last cheque: %s", err.Error())
	}

	emit := &EmitChequeMsg{
		Cheque: cheque,
	}

	// reset balance;
	err = s.resetBalance(peer, int64(cheque.Amount))
	if err != nil {
		return err
	}

	return swapPeer.Send(context.Background(), emit)
}

// createCheque creates a new cheque whose beneficiary will be the peer and
// whose serial and amount are set based on the last cheque and current balance for this peer
// The cheque will be signed and point to the issuer's contract
func (s *Swap) createCheque(peer enode.ID) (*Cheque, error) {
	var cheque *Cheque
	var err error

	swapPeer, ok := s.getPeer(peer)
	if !ok {
		return nil, fmt.Errorf("error while getting peer: %s", peer)
	}
	beneficiary := swapPeer.beneficiary

	peerBalance := s.balances[peer]
	// the balance should be negative here, we take the absolute value:
	honey := uint64(-peerBalance)

	var amount uint64
	amount, err = s.oracle.GetPrice(honey)
	if err != nil {
		return nil, fmt.Errorf("error getting price from oracle: %s", err.Error())
	}

	// we need to ignore the error check when loading from the StateStore,
	// as an error might indicate that there is no existing cheque, which
	// could mean it's the first interaction, which is absolutely valid
	err = s.loadLastSentCheque(peer)
	if err != nil && err != state.ErrNotFound {
		return nil, err
	}
	lastCheque := s.cheques[peer]

	serial := uint64(1)
	if lastCheque != nil {
		cheque = &Cheque{
			ChequeParams: ChequeParams{
				Serial: lastCheque.Serial + serial,
				Amount: lastCheque.Amount + amount,
			},
		}
	} else {
		cheque = &Cheque{
			ChequeParams: ChequeParams{
				Serial: serial,
				Amount: amount,
			},
		}
	}
	cheque.ChequeParams.Timeout = defaultCashInDelay
	cheque.ChequeParams.Contract = s.owner.Contract
	cheque.ChequeParams.Honey = honey
	cheque.Beneficiary = beneficiary

	cheque.Signature, err = s.signContent(cheque)

	return cheque, err
}

// Balance returns the balance for a given peer
func (s *Swap) Balance(peer enode.ID) (int64, error) {
	var err error
	peerBalance, ok := s.balances[peer]
	if !ok {
		err = s.store.Get(balanceKey(peer), &peerBalance)
	}
	return peerBalance, err
}

// Balances returns the balances for all known SWAP peers
func (s *Swap) Balances() (map[enode.ID]int64, error) {
	balances := make(map[enode.ID]int64)

	for peerID, peerBalance := range s.balances {
		balances[peerID] = peerBalance
	}

	// add store balances, if peer was not already added
	balanceIterFunction := func(key []byte, value []byte) (stop bool, err error) {
		peerID := keyToID(string(key), balancePrefix)
		if _, peerHasBalance := balances[peerID]; !peerHasBalance {
			var peerBalance int64
			err = json.Unmarshal(value, &peerBalance)
			if err == nil {
				balances[peerID] = peerBalance
			}
		}
		return stop, err
	}
	err := s.store.Iterate(balancePrefix, balanceIterFunction)
	if err != nil {
		return nil, err
	}

	return balances, nil
}

// loadLastSentCheque loads the last cheque for a peer from the state store (persisted)
func (s *Swap) loadLastSentCheque(peer enode.ID) (err error) {
	//only load if the current instance doesn't already have this peer's
	//last cheque in memory
	var cheque *Cheque
	if _, ok := s.cheques[peer]; !ok {
		err = s.store.Get(sentChequeKey(peer), &cheque)
		if err == nil {
			s.cheques[peer] = cheque
		}
	}
	return err
}

// loadLastReceivedCheque gets the last received cheque for the peer
// cheque gets loaded from database if not already in memory
func (s *Swap) loadLastReceivedCheque(p *Peer) (cheque *Cheque) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if p.lastReceivedCheque != nil {
		return p.lastReceivedCheque
	}
	s.store.Get(receivedChequeKey(p.ID()), &cheque)
	return
}

// saveLastReceivedCheque saves cheque as the last received cheque for peer
func (s *Swap) saveLastReceivedCheque(p *Peer, cheque *Cheque) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	p.lastReceivedCheque = cheque
	return s.store.Put(receivedChequeKey(p.ID()), cheque)
}

// Close cleans up swap
func (s *Swap) Close() {
	s.store.Close()
}

// resetBalance is called:
// * for the creditor: upon receiving the cheque
// * for the debitor: after sending the cheque
func (s *Swap) resetBalance(peerID enode.ID, amount int64) error {
	log.Debug("resetting balance for peer", "peer", peerID.String(), "amount", amount)
	_, err := s.updateBalance(peerID, amount)
	return err
}

// signContent signs the cheque with the owners private key
func (s *Swap) signContent(cheque *Cheque) ([]byte, error) {
	return cheque.Sign(s.owner.privateKey)
}

// GetParams returns contract parameters (Bin, ABI) from the contract
func (s *Swap) GetParams() *swap.Params {
	return s.contract.ContractParams()
}

// Deploy deploys a new swap contract
func (s *Swap) Deploy(ctx context.Context, backend swap.Backend, path string) error {
	return s.deploy(ctx, backend, path)
}

// verifyContract checks if the bytecode found at address matches the expected bytecode
func (s *Swap) verifyContract(ctx context.Context, address common.Address) error {
	return contract.ValidateCode(ctx, s.backend, address)
}

// getContractOwner retrieve the owner of the chequebook at address from the blockchain
func (s *Swap) getContractOwner(ctx context.Context, address common.Address) (common.Address, error) {
	contr, err := contract.InstanceAt(address, s.backend)
	if err != nil {
		return common.Address{}, err
	}

	return contr.Issuer(nil)
}

// deploy deploys the Swap contract
func (s *Swap) deploy(ctx context.Context, backend swap.Backend, path string) error {
	opts := bind.NewKeyedTransactor(s.owner.privateKey)
	// initial topup value
	opts.Value = big.NewInt(int64(s.params.InitialDepositAmount))
	opts.Context = ctx

	log.Info("deploying new swap", "owner", opts.From.Hex())
	address, err := s.deployLoop(opts, backend, s.owner.address, defaultHarddepositTimeoutDuration)
	if err != nil {
		log.Error("unable to deploy swap", "error", err)
		return err
	}
	s.owner.Contract = address
	log.Info("swap deployed", "address", address.Hex(), "owner", opts.From.Hex())

	return err
}

// deployLoop repeatedly tries to deploy the swap contract .
func (s *Swap) deployLoop(opts *bind.TransactOpts, backend swap.Backend, owner common.Address, defaultHarddepositTimeoutDuration time.Duration) (addr common.Address, err error) {
	var tx *types.Transaction
	for try := 0; try < deployRetries; try++ {
		if try > 0 {
			time.Sleep(deployDelay)
		}

		if _, s.contract, tx, err = contract.Deploy(opts, backend, owner, defaultHarddepositTimeoutDuration); err != nil {
			log.Warn("can't send chequebook deploy tx", "try", try, "error", err)
			continue
		}
		if addr, err = bind.WaitDeployed(opts.Context, backend, tx); err != nil {
			log.Warn("chequebook deploy error", "try", try, "error", err)
			continue
		}
		return addr, nil
	}
	return addr, err
}
