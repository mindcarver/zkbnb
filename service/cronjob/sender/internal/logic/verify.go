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
 */

package logic

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	zecreyLegend "github.com/zecrey-labs/zecrey-eth-rpc/zecrey/core/zecrey-legend"
	"github.com/zecrey-labs/zecrey-legend/common/model/block"
	"github.com/zecrey-labs/zecrey-legend/common/util"

	"github.com/zeromicro/go-zero/core/logx"
)

func SendVerifiedAndExecutedBlocks(
	param *SenderParam,
	l1TxSenderModel L1TxSenderModel,
	blockModel BlockModel,
	proofSenderModel ProofSenderModel,
) (err error) {
	var (
		cli                  = param.Cli
		authCli              = param.AuthCli
		zecreyLegendInstance = param.ZecreyLegendInstance
		gasPrice             = param.GasPrice
		gasLimit             = param.GasLimit
		maxBlockCount        = param.MaxBlocksCount
		maxWaitingTime       = param.MaxWaitingTime
	)
	// scan l1 tx sender table for handled verified and executed height
	lastHandledBlock, getHandleErr := l1TxSenderModel.GetLatestHandledBlock(VerifyAndExecuteTxType)
	if getHandleErr != nil && getHandleErr != ErrNotFound {
		logx.Errorf("[SendVerifiedAndExecutedBlocks] GetLatestHandledBlock err:%v", getHandleErr)
		return err
	}
	// scan l1 tx sender table for pending verified and executed height that higher than the latest handled height
	pendingSender, getPendingerr := l1TxSenderModel.GetLatestPendingBlock(VerifyAndExecuteTxType)
	if getPendingerr != nil && err != ErrNotFound {
		logx.Errorf("[SendVerifiedAndExecutedBlocks] GetLatestPendingBlock err:%v", getPendingerr)
		return err
	}
	// case 1: check tx status on L1
	if getHandleErr == ErrNotFound && getPendingerr == nil {
		_, isPending, err := cli.GetTransactionByHash(pendingSender.L1TxHash)
		// if err != nil, means we cannot get this tx by hash
		if err != nil {
			// if we cannot get it from rpc and the time over 1 min
			lastUpdatedAt := pendingSender.UpdatedAt.UnixMilli()
			now := time.Now().UnixMilli()
			if now-lastUpdatedAt > maxWaitingTime {
				// drop the record
				err := l1TxSenderModel.DeleteL1TxSender(pendingSender)
				if err != nil {
					logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to delete l1 tx sender: %s", err.Error())
					return err
				}
				return nil
			} else {
				return nil
			}
		}
		// if it is pending, still waiting
		if isPending {
			logx.Infof("[SendVerifiedAndExecutedBlocks] tx is still pending, no need to work for anything tx hash: %s", pendingSender.L1TxHash)
			return nil
		} else {
			receipt, err := cli.GetTransactionReceipt(pendingSender.L1TxHash)
			if err != nil {
				logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to get transaction receipt: %s", err.Error())
				return err
			}
			if receipt.Status == 0 {
				logx.Infof("[SendVerifiedAndExecutedBlocks] the transaction is failure, please check: %s", pendingSender.L1TxHash)
				return nil
			}
		}
	}
	// case 2:
	if getHandleErr == nil && getPendingerr == nil {
		isSuccess, err := cli.WaitingTransactionStatus(pendingSender.L1TxHash)
		// if err != nil, means we cannot get this tx by hash
		if err != nil {
			// if we cannot get it from rpc and the time over 1 min
			lastUpdatedAt := pendingSender.UpdatedAt.UnixMilli()
			if time.Now().UnixMilli()-lastUpdatedAt > maxWaitingTime {
				// drop the record
				if err := l1TxSenderModel.DeleteL1TxSender(pendingSender); err != nil {
					logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to delete l1 tx sender: %s", err.Error())
					return err
				}
			}
			return nil
		}
		// if it is pending, still waiting
		if !isSuccess {
			return nil
		}
	}
	// case 3:  means we haven't verified and executed new blocks, just start to commit
	var (
		start                         int64
		blocks                        []*block.Block
		pendingVerifyAndExecuteBlocks []ZecreyLegendVerifyBlockInfo
	)
	if getHandleErr == ErrNotFound && getPendingerr == ErrNotFound {
		// get blocks from block table
		blocks, err = blockModel.GetBlocksForProverBetween(1, int64(maxBlockCount))
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] GetBlocksForProverBetween err:%v, maxBlockCount:%v", err, maxBlockCount)
			return err
		}
		pendingVerifyAndExecuteBlocks, err = ConvertBlocksToVerifyAndExecuteBlockInfos(blocks)
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to convert blocks to verify and execute block infos: %s", err.Error())
			return err
		}
		start = int64(1)
	}
	if getHandleErr == nil && getPendingerr == ErrNotFound {
		blocks, err = blockModel.GetBlocksForProverBetween(lastHandledBlock.L2BlockHeight+1, lastHandledBlock.L2BlockHeight+int64(maxBlockCount))
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to get sender new blocks: %s", err.Error())
			return err
		}
		pendingVerifyAndExecuteBlocks, err = ConvertBlocksToVerifyAndExecuteBlockInfos(blocks)
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to convert blocks to commit block infos: %s", err.Error())
			return err
		}
		start = lastHandledBlock.L2BlockHeight + 1
	}
	proofSenders, err := proofSenderModel.GetProofsByBlockRange(start, blocks[len(blocks)-1].BlockHeight, maxBlockCount)
	if err != nil {
		logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to get proofs: %s", err.Error())
		return err
	}
	if len(proofSenders) != len(blocks) {
		logx.Errorf("[SendVerifiedAndExecutedBlocks] we haven't generated related proofs, please wait")
		return errors.New("[SendVerifiedAndExecutedBlocks] we haven't generated related proofs, please wait")
	}
	var proofs []*big.Int
	for _, proofSender := range proofSenders {
		var proofInfo *util.FormattedProof
		err = json.Unmarshal([]byte(proofSender.ProofInfo), &proofInfo)
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] unable to unmarshal proof info: %s", err.Error())
			return err
		}
		proofs = append(proofs, proofInfo.A[:]...)
		proofs = append(proofs, proofInfo.B[0][0], proofInfo.B[0][1])
		proofs = append(proofs, proofInfo.B[1][0], proofInfo.B[1][1])
		proofs = append(proofs, proofInfo.C[:]...)
	}
	// commit blocks on-chain
	if len(pendingVerifyAndExecuteBlocks) != 0 {
		txHash, err := zecreyLegend.VerifyAndExecuteBlocks(cli, authCli, zecreyLegendInstance,
			pendingVerifyAndExecuteBlocks, proofs, gasPrice, gasLimit)
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] VerifyAndExecuteBlocks err:%v", err)
			return err
		}
		// update l1 tx sender table records
		newSender := &L1TxSender{
			L1TxHash:      txHash,
			TxStatus:      PendingStatus,
			TxType:        VerifyAndExecuteTxType,
			L2BlockHeight: int64(pendingVerifyAndExecuteBlocks[len(pendingVerifyAndExecuteBlocks)-1].BlockHeader.BlockNumber),
		}
		isValid, err := l1TxSenderModel.CreateL1TxSender(newSender)
		if err != nil {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] CreateL1TxSender err:%v", err)
			return err
		}
		if !isValid {
			logx.Errorf("[SendVerifiedAndExecutedBlocks] cannot create new senders")
			return errors.New("[SendVerifiedAndExecutedBlocks] cannot create new senders")
		}
		logx.Errorf("[SendVerifiedAndExecutedBlocks] new blocks have been verified and executed(height): %v", newSender.L2BlockHeight)
		return nil
	}
	return nil
}
