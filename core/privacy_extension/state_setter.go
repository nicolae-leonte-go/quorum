package privacy_extension

import (
	"encoding/base64"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	extension "github.com/ethereum/go-ethereum/contract-extension/contractExtensionContracts"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/private"
)

var DefaultExtensionHandler = NewExtensionHandler(private.P)

type ExtensionHandler struct {
	ptm 			private.PrivateTransactionManager
}

func NewExtensionHandler(transactionManager private.PrivateTransactionManager) *ExtensionHandler {
	return &ExtensionHandler{ptm: transactionManager}
}

func (handler *ExtensionHandler) CheckExtensionAndSetPrivateState(txLogs []*types.Log, privateState *state.StateDB) {
	// there should be two logs,
	// the first being the state extension log, the second being the event finished log
	if len(txLogs) != 2 {
		//not an extension transaction, so don't check
		return
	}

	if txLog := txLogs[0]; logContainsExtensionTopic(txLog) {
		//this is a direct state share
		hash, uuid, err := extension.UnpackStateSharedLog(txLog.Data)
		if err != nil {
			return
		}
		accounts, found := handler.FetchStateData(hash, uuid)
		if !found {
			return
		}
		snapshotId := privateState.Snapshot()
		if success := setState(privateState, accounts); !success {
			privateState.RevertToSnapshot(snapshotId)
		}
	}
}

func (handler *ExtensionHandler) FetchStateData(hash string, uuid string) (map[string]extension.AccountWithMetadata, bool) {
	if uuidIsSentByUs := handler.UuidIsOwn(uuid); !uuidIsSentByUs {
		return nil, false
	}

	stateData, ok := handler.FetchDataFromPTM(hash)
	if !ok {
		//there is nothing to do here, the state wasn't shared with us
		log.Info("Extension", "No state shared with us")
		return nil, false
	}

	var accounts map[string]extension.AccountWithMetadata
	if err := json.Unmarshal(stateData, &accounts); err != nil {
		log.Info("Extension", "Could not unmarshal data")
		return nil, false
	}
	return accounts, true
}

// Checks

func (handler *ExtensionHandler) FetchDataFromPTM(hash string) ([]byte, bool){
	ptmHash, _ := base64.StdEncoding.DecodeString(hash)
	stateData, err := handler.ptm.Receive(ptmHash)

	if stateData == nil || err != nil {
		return nil, false
	}
	return stateData, true
}

func (handler *ExtensionHandler) UuidIsOwn(uuid string) bool {
	if uuid == "" {
		//we never called accept
		log.Info("Extension", "State shared by accept never called")
		return false
	}
	encryptedTxHash := common.BytesToEncryptedPayloadHash(common.FromHex(uuid))

	isSender, err := handler.ptm.IsSender(encryptedTxHash)
	if err != nil {
		log.Warn("Extension: could not determine if we are sender", "err", err.Error())
		return false
	}
	return isSender
}