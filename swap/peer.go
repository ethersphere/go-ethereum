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

package swap

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethersphere/swarm/p2p/protocols"
)

// ErrDontOwe indictates that no balance is actially owned
var ErrDontOwe = errors.New("no negative balance")

// Peer is a devp2p peer for the Swap protocol
type Peer struct {
	*protocols.Peer
	lock               sync.RWMutex
	swap               *Swap
	beneficiary        common.Address
	contractAddress    common.Address
	lastReceivedCheque *Cheque
	lastSentCheque     *Cheque
	balance            int64
	logger             log.Logger // logger for swap related messages and audit trail with peer identifier
}

// NewPeer creates a new swap Peer instance
func NewPeer(p *protocols.Peer, s *Swap, beneficiary common.Address, contractAddress common.Address) (peer *Peer, err error) {
	peer = &Peer{
		Peer:            p,
		swap:            s,
		beneficiary:     beneficiary,
		contractAddress: contractAddress,
		logger:          newPeerLogger(s, p.ID()),
	}

	if peer.lastReceivedCheque, err = s.loadLastReceivedCheque(p.ID()); err != nil {
		return nil, err
	}

	if peer.lastSentCheque, err = s.loadLastSentCheque(p.ID()); err != nil {
		return nil, err
	}

	if peer.balance, err = s.loadBalance(p.ID()); err != nil {
		return nil, err
	}
	return peer, nil
}

func (p *Peer) getLastReceivedCheque() *Cheque {
	return p.lastReceivedCheque
}

func (p *Peer) getLastSentCheque() *Cheque {
	return p.lastSentCheque
}

func (p *Peer) setLastReceivedCheque(cheque *Cheque) error {
	p.lastReceivedCheque = cheque
	return p.swap.saveLastReceivedCheque(p.ID(), cheque)
}

func (p *Peer) setLastSentCheque(cheque *Cheque) error {
	p.lastSentCheque = cheque
	return p.swap.saveLastSentCheque(p.ID(), cheque)
}

func (p *Peer) getLastSentCumulativePayout() uint64 {
	lastCheque := p.getLastSentCheque()
	if lastCheque != nil {
		return lastCheque.CumulativePayout
	}
	return 0
}

func (p *Peer) setBalance(balance int64) error {
	p.balance = balance
	return p.swap.saveBalance(p.ID(), balance)
}

func (p *Peer) getBalance() int64 {
	return p.balance
}

// To be called with mutex already held
func (p *Peer) updateBalance(amount int64) error {
	//adjust the balance
	//if amount is negative, it will decrease, otherwise increase
	newBalance := p.getBalance() + amount
	if err := p.setBalance(newBalance); err != nil {
		return err
	}
	p.logger.Debug("updated balance", "balance", strconv.FormatInt(newBalance, 10))
	return nil
}

// createCheque creates a new cheque whose beneficiary will be the peer and
// whose amount is based on the last cheque and current balance for this peer
// The cheque will be signed and point to the issuer's contract
// To be called with mutex already held
// Caller must be careful that the same resources aren't concurrently read and written by multiple routines
func (p *Peer) createCheque() (*Cheque, error) {
	var cheque *Cheque
	var err error

	if p.getBalance() >= 0 {
		return nil, fmt.Errorf("expected negative balance, found: %d", p.getBalance())
	}
	// the balance should be negative here, we take the absolute value:
	honey := uint64(-p.getBalance())

	amount, err := p.swap.honeyPriceOracle.GetPrice(honey)
	if err != nil {
		return nil, fmt.Errorf("error getting price from oracle: %v", err)
	}

	total := p.getLastSentCumulativePayout()

	cheque = &Cheque{
		ChequeParams: ChequeParams{
			CumulativePayout: total + amount,
			Contract:         p.swap.GetParams().ContractAddress,
			Beneficiary:      p.beneficiary,
		},
		Honey: honey,
	}
	cheque.Signature, err = cheque.Sign(p.swap.owner.privateKey)

	return cheque, err
}

// sendCheque sends a cheque to peer
// To be called with mutex already held
// Caller must be careful that the same resources aren't concurrently read and written by multiple routines
func (p *Peer) sendCheque() error {
	cheque, err := p.createCheque()
	if err != nil {
		return fmt.Errorf("error while creating cheque: %v", err)
	}

	if err := p.setLastSentCheque(cheque); err != nil {
		return fmt.Errorf("error while storing the last cheque: %v", err)
	}

	if err := p.updateBalance(int64(cheque.Honey)); err != nil {
		return err
	}

	p.logger.Info("sending cheque to peer", "honey", cheque.Honey, "cumulativePayout", cheque.ChequeParams.CumulativePayout, "beneficiary", cheque.Beneficiary, "contract", cheque.Contract)
	return p.Send(context.Background(), &EmitChequeMsg{
		Cheque: cheque,
	})
}
