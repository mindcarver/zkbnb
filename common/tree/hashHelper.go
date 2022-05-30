/*
 * Copyright © 2021 Zecrey Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package tree

import (
	"bytes"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zecrey-labs/zecrey-legend/common/util"
	"github.com/zeromicro/go-zero/core/logx"
)

func ComputeAccountLeafHash(
	accountNameHash string,
	pk string,
	nonce int64,
	collectionNonce int64,
	assetRoot []byte,
) (hashVal []byte, err error) {
	hFunc := mimc.NewMiMC()
	var buf bytes.Buffer
	buf.Write(common.FromHex(accountNameHash))
	err = util.WritePkIntoBuf(&buf, pk)
	if err != nil {
		logx.Errorf("[ComputeAccountAssetLeafHash] unable to write pk into buf: %s", err.Error())
		return nil, err
	}
	util.WriteInt64IntoBuf(&buf, nonce)
	util.WriteInt64IntoBuf(&buf, collectionNonce)
	buf.Write(assetRoot)
	hFunc.Reset()
	hFunc.Write(buf.Bytes())
	hashVal = hFunc.Sum(nil)
	return hashVal, nil
}

func ComputeAccountAssetLeafHash(
	balance string,
	lpAmount string,
	offerCanceledOrFinalized string,
) (hashVal []byte, err error) {
	hFunc := mimc.NewMiMC()
	var buf bytes.Buffer
	err = util.WriteStringBigIntIntoBuf(&buf, balance)
	if err != nil {
		logx.Errorf("[ComputeAccountAssetLeafHash] invalid balance: %s", err.Error())
		return nil, err
	}
	err = util.WriteStringBigIntIntoBuf(&buf, lpAmount)
	if err != nil {
		logx.Errorf("[ComputeAccountAssetLeafHash] invalid balance: %s", err.Error())
		return nil, err
	}
	err = util.WriteStringBigIntIntoBuf(&buf, offerCanceledOrFinalized)
	if err != nil {
		logx.Errorf("[ComputeAccountAssetLeafHash] invalid balance: %s", err.Error())
		return nil, err
	}
	hFunc.Write(buf.Bytes())
	return hFunc.Sum(nil), nil
}

func ComputeLiquidityAssetLeafHash(
	assetAId int64,
	assetA string,
	assetBId int64,
	assetB string,
	lpAmount string,
	kLast string,
	feeRate int64,
	treasuryAccountIndex int64,
	treasuryRate int64,
) (hashVal []byte, err error) {
	hFunc := mimc.NewMiMC()
	var buf bytes.Buffer
	util.WriteInt64IntoBuf(&buf, assetAId)
	err = util.WriteStringBigIntIntoBuf(&buf, assetA)
	if err != nil {
		logx.Errorf("[ComputeLiquidityAssetLeafHash] unable to write big int to buf: %s", err.Error())
		return nil, err
	}
	util.WriteInt64IntoBuf(&buf, assetBId)
	err = util.WriteStringBigIntIntoBuf(&buf, assetB)
	if err != nil {
		logx.Errorf("[ComputeLiquidityAssetLeafHash] unable to write big int to buf: %s", err.Error())
		return nil, err
	}
	err = util.WriteStringBigIntIntoBuf(&buf, lpAmount)
	if err != nil {
		logx.Errorf("[ComputeLiquidityAssetLeafHash] unable to write big int to buf: %s", err.Error())
		return nil, err
	}
	err = util.WriteStringBigIntIntoBuf(&buf, kLast)
	if err != nil {
		logx.Errorf("[ComputeLiquidityAssetLeafHash] unable to write big int to buf: %s", err.Error())
		return nil, err
	}
	util.WriteInt64IntoBuf(&buf, feeRate)
	util.WriteInt64IntoBuf(&buf, treasuryAccountIndex)
	util.WriteInt64IntoBuf(&buf, treasuryRate)
	hFunc.Write(buf.Bytes())
	hashVal = hFunc.Sum(nil)
	return hashVal, nil
}

func ComputeNftAssetLeafHash(
	creatorAccountIndex int64,
	ownerAccountIndex int64,
	nftContentHash string,
	nftL1Address string,
	nftL1TokenId string,
	creatorTreasuryRate int64,
) (hashVal []byte, err error) {
	hFunc := mimc.NewMiMC()
	var buf bytes.Buffer
	util.WriteInt64IntoBuf(&buf, creatorAccountIndex)
	util.WriteInt64IntoBuf(&buf, ownerAccountIndex)
	buf.Write(common.FromHex(nftContentHash))
	err = util.WriteAddressIntoBuf(&buf, nftL1Address)
	if err != nil {
		logx.Errorf("[ComputeNftAssetLeafHash] unable to write address to buf: %s", err.Error())
		return nil, err
	}
	err = util.WriteStringBigIntIntoBuf(&buf, nftL1TokenId)
	if err != nil {
		logx.Errorf("[ComputeNftAssetLeafHash] unable to write big int to buf: %s", err.Error())
		return nil, err
	}
	util.WriteInt64IntoBuf(&buf, creatorTreasuryRate)
	hFunc.Write(buf.Bytes())
	hashVal = hFunc.Sum(nil)
	return hashVal, nil
}

func ComputeStateRootHash(
	accountRoot []byte,
	liquidityRoot []byte,
	nftRoot []byte,
) []byte {
	hFunc := mimc.NewMiMC()
	hFunc.Write(accountRoot)
	hFunc.Write(liquidityRoot)
	hFunc.Write(nftRoot)
	return hFunc.Sum(nil)
}