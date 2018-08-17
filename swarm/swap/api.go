// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package swap

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var (
	ErrNoSuchPeerAccounting = errors.New("No accounting with that peer")
)

// Wrapper for receiving pss messages when using the pss API
// providing access to sender of message
type APIMsg struct {
	Msg hexutil.Bytes
}

// Additional public methods accessible through API for pss
type API struct {
	*SwapProtocol
}

//TODO: define metrics
type SwapMetrics struct {
}

func NewAPI(swap *SwapProtocol) *API {
	return &API{SwapProtocol: swap}
}

func (swapapi *API) BalanceWithPeer(ctx context.Context, peer discover.NodeID) (balance *big.Int, err error) {
	balance = swapapi.swap.peers[peer].balance
	if balance == nil {
		err = ErrNoSuchPeerAccounting
	}
	return
}

func (swapapi *API) Balance(ctx context.Context) (balance *big.Int, err error) {
	balance = big.NewInt(0)
	for _, peer := range swapapi.swap.peers {
		balance.Add(balance, peer.balance)
	}
	return
}

func (swapapi *API) GetSwapMetrics() (*SwapMetrics, error) {
	return nil, nil
}
