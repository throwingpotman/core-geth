// Copyright 2019 The multi-geth Authors
// This file is part of the multi-geth library.
//
// The multi-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The multi-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the multi-geth library. If not, see <http://www.gnu.org/licenses/>.


package generic

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/goethereum"
	"github.com/ethereum/go-ethereum/params/types/multigeth"
	"github.com/ethereum/go-ethereum/params/types/parity"
	"github.com/ethereum/go-ethereum/params/vars"
	"github.com/tidwall/gjson"
)

// GenericCC is a generic-y struct type used to expose some meta-logic methods
// shared by all ChainConfigurator implementations but not existing in that interface.
// These logics differentiate from the logics present in the ChainConfigurator interface
// itself because they are chain-aware, or fit nuanced, or adhoc, use cases and should
// not be demanded of EVM-based ecosystem logic as a whole. Debatable. NOTE.
type GenericCC struct {
	ctypes.ChainConfigurator
}

func AsGenericCC(c ctypes.ChainConfigurator) GenericCC {
	return GenericCC{c}
}

func (c GenericCC) DAOSupport() bool {
	if gc, ok := c.ChainConfigurator.(*goethereum.ChainConfig); ok {
		return gc.DAOForkSupport
	}
	if mg, ok := c.ChainConfigurator.(*multigeth.MultiGethChainConfig); ok {
		return mg.GetEthashEIP779Transition() != nil
	}
	if pc, ok := c.ChainConfigurator.(*parity.ParityChainSpec); ok {
		return pc.Engine.Ethash.Params.DaoHardforkTransition != nil &&
			pc.Engine.Ethash.Params.DaoHardforkBeneficiary != nil &&
			*pc.Engine.Ethash.Params.DaoHardforkBeneficiary == vars.DAORefundContract &&
			len(pc.Engine.Ethash.Params.DaoHardforkAccounts) == len(vars.DAODrainList())
	}
	panic(fmt.Sprintf("uimplemented DAO logic, config: %v", c.ChainConfigurator))
}

// Following vars define sufficient JSON schema keys for configurator type inference.
var (
	paritySchemaKeysMust = []string{
		"engine",
		"genesis.seal",
	}
	multigethSchemaMust = []string{
		"networkId", "config.networkId",
		"eip2FBlock", "config.eip2FBlock",
	}
	goethereumSchemaMust = []string{
		"difficulty",
		"chainId", "config.chainId",
		"eip158Block", "config.eip158Block",
		"byzantiumBlock", "config.byzantiumBlock",
	}
)

func UnmarshalChainConfigurator(input []byte) (ctypes.ChainConfigurator, error) {
	var cases = map[ctypes.ChainConfigurator][]string{
		&parity.ParityChainSpec{}: paritySchemaKeysMust,
		&multigeth.MultiGethChainConfig{}: multigethSchemaMust,
		&goethereum.ChainConfig{}: goethereumSchemaMust,
	}
	for c, fn := range cases {
		ok, err := asMapHasAnyKey(input, fn)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := json.Unmarshal(input, c); err != nil {
				return nil, err
			}
			return c, nil
		}
	}
	return nil, errors.New("invalid configurator schema")
}

func asMapHasAnyKey(input []byte, keys []string) (bool, error) {
	results := gjson.GetManyBytes(input, keys...)
	for _, g := range results {
		if g.Exists() {
			return true, nil
		}
	}
	return false, nil
}