package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	curve "github.com/zecrey-labs/zecrey-crypto/ecc/ztwistededwards/tebn254"
	"github.com/zecrey-labs/zecrey-crypto/wasm/zecrey-legend/legendTxTypes"
	"github.com/zecrey-labs/zecrey-legend/common/commonTx"
	"github.com/zecrey-labs/zecrey-legend/common/util"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/globalRPC/globalRPCProto"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/globalRPC/internal/config"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/globalRPC/internal/server"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/globalRPC/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"math/big"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestSendTransferNftTx(t *testing.T) {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	logx.MustSetup(c.LogConf)
	ctx := svc.NewServiceContext(c)

	/*
		err := globalmapHandler.ReloadGlobalMap(ctx)
		if err != nil {
			logx.Error("[main] %s", err.Error())
			return
		}
	*/

	srv := server.NewGlobalRPCServer(ctx)
	txInfo := constructSendTransferNftTxInfo()
	resp, err := srv.SendTx(
		context.Background(),
		&globalRPCProto.ReqSendTx{
			TxType: commonTx.TxTypeTransferNft,
			TxInfo: txInfo,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(respBytes))
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
}

func constructSendTransferNftTxInfo() string {
	// from sher.legend to gavin.legend
	//seed := "28e1a3762ff9944e9a4ad79477b756ef0aff3d2af76f0f40a0c3ec6ca76cf24b"
	seed := "17673b9a9fdec6dc90c7cc1eb1c939134dfb659d2f08edbe071e5c45f343d008"
	key, err := curve.GenerateEddsaPrivateKey(seed)
	if err != nil {
		panic(err)
	}
	nameHash, err := util.AccountNameHash("sher.legend")
	if err != nil {
		panic(err)
	}
	expiredAt := time.Now().Add(time.Hour * 2).UnixMilli()
	txInfo := &commonTx.TransferNftTxInfo{
		FromAccountIndex:  3,
		ToAccountIndex:    2,
		ToAccountNameHash: nameHash,
		NftIndex:          1,
		GasAccountIndex:   1,
		GasFeeAssetId:     0,
		GasFeeAssetAmount: big.NewInt(5000),
		CallData:          "",
		CallDataHash:      nil,
		ExpiredAt:         expiredAt,
		Nonce:             1,
		Sig:               nil,
	}
	hFunc := mimc.NewMiMC()
	hFunc.Write([]byte(txInfo.CallData))
	callDataHash := hFunc.Sum(nil)
	txInfo.CallDataHash = callDataHash
	hFunc.Reset()
	msgHash, err := legendTxTypes.ComputeTransferNftMsgHash(txInfo, hFunc)
	if err != nil {
		panic(err)
	}
	hFunc.Reset()
	signature, err := key.Sign(msgHash, hFunc)
	if err != nil {
		panic(err)
	}
	txInfo.Sig = signature
	txInfoBytes, err := json.Marshal(txInfo)
	if err != nil {
		panic(err)
	}
	return string(txInfoBytes)
}