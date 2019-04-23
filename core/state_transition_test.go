package core

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/private"
	"github.com/ethereum/go-ethereum/private/engine"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/ethereum/go-ethereum/common/math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	testifyassert "github.com/stretchr/testify/assert"
)

var (
	c1 = &contract{
		name:     "c1",
		abi:      mustParse(c1AbiDefinition),
		bytecode: common.Hex2Bytes("608060405234801561001057600080fd5b506040516020806105a88339810180604052602081101561003057600080fd5b81019080805190602001909291905050508060008190555050610550806100586000396000f3fe608060405260043610610051576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b1146100565780636d4ce63c146100a5578063d7139463146100d0575b600080fd5b34801561006257600080fd5b5061008f6004803603602081101561007957600080fd5b810190808035906020019092919050505061010b565b6040518082815260200191505060405180910390f35b3480156100b157600080fd5b506100ba61011e565b6040518082815260200191505060405180910390f35b3480156100dc57600080fd5b50610109600480360360208110156100f357600080fd5b8101908080359060200190929190505050610127565b005b6000816000819055506000549050919050565b60008054905090565b600030610132610212565b808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050604051809103906000f080158015610184573d6000803e3d6000fd5b5090508073ffffffffffffffffffffffffffffffffffffffff166360fe47b1836040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050600060405180830381600087803b1580156101f657600080fd5b505af115801561020a573d6000803e3d6000fd5b505050505050565b604051610302806102238339019056fe608060405234801561001057600080fd5b506040516020806103028339810180604052602081101561003057600080fd5b8101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050610271806100916000396000f3fe608060405260043610610046576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b11461004b5780636d4ce63c14610086575b600080fd5b34801561005757600080fd5b506100846004803603602081101561006e57600080fd5b81019080803590602001909291905050506100b1565b005b34801561009257600080fd5b5061009b610180565b6040518082815260200191505060405180910390f35b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166360fe47b1826040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050602060405180830381600087803b15801561014157600080fd5b505af1158015610155573d6000803e3d6000fd5b505050506040513d602081101561016b57600080fd5b81019080805190602001909291905050505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636d4ce63c6040518163ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040160206040518083038186803b15801561020557600080fd5b505afa158015610219573d6000803e3d6000fd5b505050506040513d602081101561022f57600080fd5b810190808051906020019092919050505090509056fea165627a7a72305820a537f4c360ce5c6f55523298e314e6456e5c3e02c170563751dfda37d3aeddb30029a165627a7a7230582060396bfff29d2dfc5a9f4216bfba5e24d031d54fd4b26ebebde1a26c59df0c1e0029"),
	}
	c2 = &contract{
		name:     "c2",
		abi:      mustParse(c2AbiDefinition),
		bytecode: common.Hex2Bytes("608060405234801561001057600080fd5b506040516020806102f58339810180604052602081101561003057600080fd5b8101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050610264806100916000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b1146100585780636d4ce63c14610086575b600080fd5b6100846004803603602081101561006e57600080fd5b81019080803590602001909291905050506100a4565b005b61008e610173565b6040518082815260200191505060405180910390f35b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166360fe47b1826040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050602060405180830381600087803b15801561013457600080fd5b505af1158015610148573d6000803e3d6000fd5b505050506040513d602081101561015e57600080fd5b81019080805190602001909291905050505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636d4ce63c6040518163ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040160206040518083038186803b1580156101f857600080fd5b505afa15801561020c573d6000803e3d6000fd5b505050506040513d602081101561022257600080fd5b810190808051906020019092919050505090509056fea165627a7a72305820dd8a5dcf693e1969289c444a282d0684a9760bac26f1e4e0139d46821ec1979b0029"),
	}
)

type contract struct {
	abi      abi.ABI
	bytecode []byte
	name     string
}

func (c *contract) create(args ...interface{}) []byte {
	bytes, err := c.abi.Pack("", args...)
	if err != nil {
		panic("can't pack: " + err.Error())
	}
	return append(c.bytecode, bytes...)
}

func (c *contract) set(value int64) []byte {
	bytes, err := c.abi.Pack("set", big.NewInt(value))
	if err != nil {
		panic("can't pack: " + err.Error())
	}
	return bytes
}

func init() {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.StreamHandler(os.Stdout, log.TerminalFormat(true)))
}

func TestApplyMessage_Private_whenTypical(t *testing.T) {
	originalP := private.P
	defer func() { private.P = originalP }()
	mockPM := newMockPrivateTransactionManager()
	private.P = mockPM
	assert := testifyassert.New(t)
	cfg := newConfig().
		setPrivacyFlag(engine.PrivacyFlagLegacy).
		setData([]byte("arbitrary encrypted payload hash"))
	gp := new(GasPool).AddGas(math.MaxUint64)
	privateMsg := newTypicalPrivateMessage(cfg)

	mockPM.When("Receive").Return(c1.create(big.NewInt(42)), &engine.ExtraMetadata{
		PrivacyFlag: engine.PrivacyFlagLegacy,
	}, nil)

	_, _, fail, err := ApplyMessage(newEVM(cfg), privateMsg, gp)

	assert.NoError(err, "EVM execution")
	assert.False(fail, "Transaction receipt status")
	mockPM.Verify(assert)
}

// C1 is a existing contract before privacy enhancements implementation
func TestApplyMessage_Private_whenPartyProtectionC2InteractsExistingLegacyC1(t *testing.T) {
	originalP := private.P
	defer func() { private.P = originalP }()
	mockPM := newMockPrivateTransactionManager()
	private.P = mockPM
	assert := testifyassert.New(t)
	cfg := newConfig()

	// create c1 like c1 already exist before privacy enhancements
	c1EncPayloadHash := []byte("c1")
	cfg.setPrivacyFlag(math.MaxUint64).
		setData(c1EncPayloadHash)
	c1Address := createContract(cfg, mockPM, assert, c1, big.NewInt(42))

	// create c2
	c2EncPayloadHash := []byte("c2")
	cfg.setPrivacyFlag(engine.PrivacyFlagPartyProtection).
		setData(c2EncPayloadHash).
		setNonce(1)
	c2Address := createContract(cfg, mockPM, assert, c2, c1Address)

	// calling C2.Set()
	cfg.setPrivacyFlag(engine.PrivacyFlagPartyProtection).
		setData([]byte("arbitrary enc payload hash")).
		setNonce(2).
		setTo(c2Address)
	privateMsg := newTypicalPrivateMessage(cfg)
	mockPM.When("Receive").Return(c2.set(53), &engine.ExtraMetadata{
		ACHashes: common.EncryptedPayloadHashes{
			common.BytesToEncryptedPayloadHash(c2EncPayloadHash): struct{}{},
		},
		PrivacyFlag: engine.PrivacyFlagPartyProtection,
	}, nil)

	_, _, fail, err := ApplyMessage(newEVM(cfg), privateMsg, new(GasPool).AddGas(math.MaxUint64))

	assert.NoError(err, "EVM execution")
	assert.True(fail, "Transaction receipt status")
	mockPM.Verify(assert)
}

func createContract(cfg *config, mockPM *mockPrivateTransactionManager, assert *testifyassert.Assertions, c *contract, args ...interface{}) common.Address {
	defer mockPM.reset()

	privateMsg := newTypicalPrivateMessage(cfg)
	metadata := &engine.ExtraMetadata{}
	if cfg.privacyFlag < math.MaxUint64 {
		metadata.PrivacyFlag = cfg.privacyFlag
	}
	mockPM.When("Receive").Return(c.create(args...), metadata, nil)

	evm := newEVM(cfg)
	_, _, fail, err := ApplyMessage(evm, privateMsg, new(GasPool).AddGas(math.MaxUint64))

	assert.NoError(err, "%s: EVM execution", c.name)
	assert.False(fail, "%s: Transaction receipt status", c.name)
	mockPM.Verify(assert)
	createdContracts := evm.CreatedContracts()
	assert.Len(createdContracts, 1, "%s: Number of created contracts", c.name)
	address := createdContracts[0]
	log.Debug("Created "+c.name, "address", address)
	return address
}

func newTypicalPrivateMessage(cfg *config) PrivateMessage {
	var tx *types.Transaction
	if cfg.to == nil {
		tx = types.NewContractCreation(cfg.nonce, big.NewInt(0), math.MaxUint64, big.NewInt(0), cfg.data)
	} else {
		tx = types.NewTransaction(cfg.nonce, *cfg.to, big.NewInt(0), math.MaxUint64, big.NewInt(0), cfg.data)
	}
	tx.SetPrivate()
	if cfg.privacyFlag < math.MaxUint64 {
		tx.SetTxPrivacyMetadata(&types.PrivacyMetadata{
			PrivacyFlag: cfg.privacyFlag,
		})
	} else {
		tx.SetTxPrivacyMetadata(nil) // simulate legacy transaction
	}
	msg, err := tx.AsMessage(&stubSigner{})
	if err != nil {
		panic(fmt.Sprintf("can't create a new private message: %s", err))
	}
	cfg.currentTx = tx
	return PrivateMessage(msg)
}

type config struct {
	from  common.Address
	to    *common.Address
	data  []byte
	nonce uint64

	privacyFlag engine.PrivacyFlagType

	currentTx *types.Transaction

	publicState, privateState *state.StateDB
}

func newConfig() *config {
	database := ethdb.NewMemDatabase()
	publicState, _ := state.New(common.Hash{}, state.NewDatabase(database))
	publicState.SetPersistentEthdb(database)
	privateState, _ := state.New(common.Hash{}, state.NewDatabase(database))
	privateState.SetPersistentEthdb(database)
	return &config{
		privateState: privateState,
		publicState:  publicState,
	}
}

func (cfg *config) setPrivacyFlag(f engine.PrivacyFlagType) *config {
	cfg.privacyFlag = f
	return cfg
}

func (cfg *config) setData(bytes []byte) *config {
	cfg.data = bytes
	return cfg
}

func (cfg *config) setNonce(n uint64) *config {
	cfg.nonce = n
	return cfg
}

func (cfg *config) setTo(address common.Address) *config {
	cfg.to = &address
	return cfg
}

func newEVM(cfg *config) *vm.EVM {
	context := vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },

		Origin:      common.Address{},
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int),
		Time:        big.NewInt(time.Now().Unix()),
		Difficulty:  new(big.Int),
		GasLimit:    uint64(3450366),
		GasPrice:    big.NewInt(0),
	}
	evm := vm.NewEVM(context, cfg.publicState, cfg.privateState, &params.ChainConfig{
		ChainID:        big.NewInt(1),
		ByzantiumBlock: new(big.Int),
		HomesteadBlock: new(big.Int),
		DAOForkBlock:   new(big.Int),
		DAOForkSupport: false,
		EIP150Block:    new(big.Int),
		EIP155Block:    new(big.Int),
		EIP158Block:    new(big.Int),
		IsQuorum:       true,
	}, vm.Config{})
	evm.SetCurrentTX(cfg.currentTx)
	return evm
}

func mustParse(def string) abi.ABI {
	ret, err := abi.JSON(strings.NewReader(def))
	if err != nil {
		panic(fmt.Sprintf("Can't parse ABI def %s", err))
	}
	return ret
}

type stubSigner struct {
}

func (ss *stubSigner) Sender(tx *types.Transaction) (common.Address, error) {
	return common.StringToAddress("contract"), nil
}

func (ss *stubSigner) SignatureValues(tx *types.Transaction, sig []byte) (r, s, v *big.Int, err error) {
	panic("implement me")
}

func (ss *stubSigner) Hash(tx *types.Transaction) common.Hash {
	panic("implement me")
}

func (ss *stubSigner) Equal(types.Signer) bool {
	panic("implement me")
}

type mockPrivateTransactionManager struct {
	returns       map[string][]interface{}
	currentMethod string
	count         map[string]int
}

func (mpm *mockPrivateTransactionManager) Name() string {
	return "MockPrivateTransactionManager"
}

func (mpm *mockPrivateTransactionManager) Send(data []byte, from string, to []string, extra *engine.ExtraMetadata) (common.EncryptedPayloadHash, error) {
	panic("implement me")
}

func (mpm *mockPrivateTransactionManager) SendSignedTx(data common.EncryptedPayloadHash, to []string, extra *engine.ExtraMetadata) ([]byte, error) {
	panic("implement me")
}

func (mpm *mockPrivateTransactionManager) Receive(data common.EncryptedPayloadHash) ([]byte, *engine.ExtraMetadata, error) {
	mpm.count["Receive"]++
	values := mpm.returns["Receive"]
	var (
		r1 []byte
		r2 *engine.ExtraMetadata
		r3 error
	)
	if values[0] != nil {
		r1 = values[0].([]byte)
	}
	if values[1] != nil {
		r2 = values[1].(*engine.ExtraMetadata)
	}
	if values[2] != nil {
		r3 = values[2].(error)
	}
	return r1, r2, r3
}

func (mpm *mockPrivateTransactionManager) ReceiveRaw(data common.EncryptedPayloadHash) ([]byte, *engine.ExtraMetadata, error) {
	panic("implement me")
}

func (mpm *mockPrivateTransactionManager) When(name string) *mockPrivateTransactionManager {
	mpm.currentMethod = name
	mpm.count[name] = -1
	return mpm
}

func (mpm *mockPrivateTransactionManager) Return(values ...interface{}) {
	mpm.returns[mpm.currentMethod] = values
}

func (mpm *mockPrivateTransactionManager) Verify(assert *testifyassert.Assertions) {
	for m, c := range mpm.count {
		assert.True(c > -1, "%s has not been called", m)
	}
}

func (mpm *mockPrivateTransactionManager) reset() {
	mpm.count = make(map[string]int)
	mpm.currentMethod = ""
	mpm.returns = make(map[string][]interface{})
}

func newMockPrivateTransactionManager() *mockPrivateTransactionManager {
	return &mockPrivateTransactionManager{
		returns: make(map[string][]interface{}),
		count:   make(map[string]int),
	}
}

const (
	c1AbiDefinition = `
[
	{
		"constant": false,
		"inputs": [
			{
				"name": "newValue",
				"type": "uint256"
			}
		],
		"name": "set",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "get",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "newValue",
				"type": "uint256"
			}
		],
		"name": "newContractC2",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"name": "initVal",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "constructor"
	}
]
`
	c2AbiDefinition = `
[
	{
		"constant": false,
		"inputs": [
			{
				"name": "_val",
				"type": "uint256"
			}
		],
		"name": "set",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "get",
		"outputs": [
			{
				"name": "result",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"name": "_t",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "constructor"
	}
]
`
)
