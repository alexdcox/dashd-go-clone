// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular tests due to the following build tag.
// +build rpctest

package rpctest

import (
	"os"
	"testing"
	"time"

	"github.com/alexdcox/dashd-go/chaincfg/chainhash"
	"github.com/alexdcox/dashd-go/txscript"
	"github.com/alexdcox/dashd-go/wire"
	"github.com/alexdcox/dashutil"
)

func testSendOutputs(r *Harness, t *testing.T) {
	genSpend := func(amt dashutil.Amount) *chainhash.Hash {
		// Grab a fresh address from the wallet.
		addr, err := r.NewAddress()
		if err != nil {
			t.Fatalf("unable to get new address: %v", err)
		}

		// Next, send amt BTC to this address, spending from one of our mature
		// coinbase outputs.
		addrScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			t.Fatalf("unable to generate pkscript to addr: %v", err)
		}
		output := wire.NewTxOut(int64(amt), addrScript)
		txid, err := r.SendOutputs([]*wire.TxOut{output}, 10)
		if err != nil {
			t.Fatalf("coinbase spend failed: %v", err)
		}
		return txid
	}

	assertTxMined := func(txid *chainhash.Hash, blockHash *chainhash.Hash) {
		block, err := r.Node.GetBlock(blockHash)
		if err != nil {
			t.Fatalf("unable to get block: %v", err)
		}

		numBlockTxns := len(block.Transactions)
		if numBlockTxns < 2 {
			t.Fatalf("crafted transaction wasn't mined, block should have "+
				"at least %v transactions instead has %v", 2, numBlockTxns)
		}

		minedTx := block.Transactions[1]
		txHash := minedTx.TxHash()
		if txHash != *txid {
			t.Fatalf("txid's don't match, %v vs %v", txHash, txid)
		}
	}

	// First, generate a small spend which will require only a single
	// input.
	txid := genSpend(dashutil.Amount(5 * dashutil.SatoshiPerBitcoin))

	// Generate a single block, the transaction the wallet created should
	// be found in this block.
	blockHashes, err := r.Node.Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	assertTxMined(txid, blockHashes[0])

	// Next, generate a spend much greater than the block reward. This
	// transaction should also have been mined properly.
	txid = genSpend(dashutil.Amount(500 * dashutil.SatoshiPerBitcoin))
	blockHashes, err = r.Node.Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	assertTxMined(txid, blockHashes[0])
}

func assertConnectedTo(t *testing.T, nodeA *Harness, nodeB *Harness) {
	nodeAPeers, err := nodeA.Node.GetPeerInfo()
	if err != nil {
		t.Fatalf("unable to get nodeA's peer info")
	}

	nodeAddr := nodeB.node.config.listen
	addrFound := false
	for _, peerInfo := range nodeAPeers {
		if peerInfo.Addr == nodeAddr {
			addrFound = true
			break
		}
	}

	if !addrFound {
		t.Fatal("nodeA not connected to nodeB")
	}
}

func testTearDownAll(t *testing.T) {
	// Grab a local copy of the currently active harnesses before
	// attempting to tear them all down.
	initialActiveHarnesses := ActiveHarnesses()

	// Tear down all currently active harnesses.
	if err := TearDownAll(); err != nil {
		t.Fatalf("unable to teardown all harnesses: %v", err)
	}

	// The global testInstances map should now be fully purged with no
	// active test harnesses remaining.
	if len(ActiveHarnesses()) != 0 {
		t.Fatalf("test harnesses still active after TearDownAll")
	}

	for _, harness := range initialActiveHarnesses {
		// Ensure all test directories have been deleted.
		if _, err := os.Stat(harness.testNodeDir); err == nil {
			t.Errorf("created test datadir was not deleted.")
		}
	}
}

func testGenerateAndSubmitBlock(r *Harness, t *testing.T) {
	// Generate a few test spend transactions.
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	output := wire.NewTxOut(dashutil.SatoshiPerBitcoin, pkScript)

	const numTxns = 5
	txns := make([]*dashutil.Tx, 0, numTxns)
	for i := 0; i < numTxns; i++ {
		tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
		if err != nil {
			t.Fatalf("unable to create tx: %v", err)
		}

		txns = append(txns, dashutil.NewTx(tx))
	}

	// Now generate a block with the default block version, and a zero'd
	// out time.
	block, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

	// Ensure that all created transactions were included, and that the
	// block version was properly set to the default.
	numBlocksTxns := len(block.Transactions())
	if numBlocksTxns != numTxns+1 {
		t.Fatalf("block did not include all transactions: "+
			"expected %v, got %v", numTxns+1, numBlocksTxns)
	}
	blockVersion := block.MsgBlock().Header.Version
	if blockVersion != BlockVersion {
		t.Fatalf("block version is not default: expected %v, got %v",
			BlockVersion, blockVersion)
	}

	// Next generate a block with a "non-standard" block version along with
	// time stamp a minute after the previous block's timestamp.
	timestamp := block.MsgBlock().Header.Timestamp.Add(time.Minute)
	targetBlockVersion := int32(1337)
	block, err = r.GenerateAndSubmitBlock(nil, targetBlockVersion, timestamp)
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

	// Finally ensure that the desired block version and timestamp were set
	// properly.
	header := block.MsgBlock().Header
	blockVersion = header.Version
	if blockVersion != targetBlockVersion {
		t.Fatalf("block version mismatch: expected %v, got %v",
			targetBlockVersion, blockVersion)
	}
	if !timestamp.Equal(header.Timestamp) {
		t.Fatalf("header time stamp mismatch: expected %v, got %v",
			timestamp, header.Timestamp)
	}
}

func testGenerateAndSubmitBlockWithCustomCoinbaseOutputs(r *Harness,
	t *testing.T) {
	// Generate a few test spend transactions.
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	output := wire.NewTxOut(dashutil.SatoshiPerBitcoin, pkScript)

	const numTxns = 5
	txns := make([]*dashutil.Tx, 0, numTxns)
	for i := 0; i < numTxns; i++ {
		tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
		if err != nil {
			t.Fatalf("unable to create tx: %v", err)
		}

		txns = append(txns, dashutil.NewTx(tx))
	}

	// Now generate a block with the default block version, a zero'd out
	// time, and a burn output.
	block, err := r.GenerateAndSubmitBlockWithCustomCoinbaseOutputs(txns,
		-1, time.Time{}, []wire.TxOut{{
			Value:    0,
			PkScript: []byte{},
		}})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

	// Ensure that all created transactions were included, and that the
	// block version was properly set to the default.
	numBlocksTxns := len(block.Transactions())
	if numBlocksTxns != numTxns+1 {
		t.Fatalf("block did not include all transactions: "+
			"expected %v, got %v", numTxns+1, numBlocksTxns)
	}
	blockVersion := block.MsgBlock().Header.Version
	if blockVersion != BlockVersion {
		t.Fatalf("block version is not default: expected %v, got %v",
			BlockVersion, blockVersion)
	}

	// Next generate a block with a "non-standard" block version along with
	// time stamp a minute after the previous block's timestamp.
	timestamp := block.MsgBlock().Header.Timestamp.Add(time.Minute)
	targetBlockVersion := int32(1337)
	block, err = r.GenerateAndSubmitBlockWithCustomCoinbaseOutputs(nil,
		targetBlockVersion, timestamp, []wire.TxOut{{
			Value:    0,
			PkScript: []byte{},
		}})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

	// Finally ensure that the desired block version and timestamp were set
	// properly.
	header := block.MsgBlock().Header
	blockVersion = header.Version
	if blockVersion != targetBlockVersion {
		t.Fatalf("block version mismatch: expected %v, got %v",
			targetBlockVersion, blockVersion)
	}
	if !timestamp.Equal(header.Timestamp) {
		t.Fatalf("header time stamp mismatch: expected %v, got %v",
			timestamp, header.Timestamp)
	}
}

func testMemWalletLockedOutputs(r *Harness, t *testing.T) {
	// Obtain the initial balance of the wallet at this point.
	startingBalance := r.ConfirmedBalance()

	// First, create a signed transaction spending some outputs.
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	outputAmt := dashutil.Amount(50 * dashutil.SatoshiPerBitcoin)
	output := wire.NewTxOut(int64(outputAmt), pkScript)
	tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		t.Fatalf("unable to create transaction: %v", err)
	}

	// The current wallet balance should now be at least 50 BTC less
	// (accounting for fees) than the period balance
	currentBalance := r.ConfirmedBalance()
	if !(currentBalance <= startingBalance-outputAmt) {
		t.Fatalf("spent outputs not locked: previous balance %v, "+
			"current balance %v", startingBalance, currentBalance)
	}

	// Now unlocked all the spent inputs within the unbroadcast signed
	// transaction. The current balance should now be exactly that of the
	// starting balance.
	r.UnlockOutputs(tx.TxIn)
	currentBalance = r.ConfirmedBalance()
	if currentBalance != startingBalance {
		t.Fatalf("current and starting balance should now match: "+
			"expected %v, got %v", startingBalance, currentBalance)
	}
}

var harnessTestCases = []HarnessTestCase{
	testSendOutputs,
	testConnectNode,
	testActiveHarnesses,
	testJoinBlocks,
	testJoinMempools, // Depends on results of testJoinBlocks
	testGenerateAndSubmitBlock,
	testGenerateAndSubmitBlockWithCustomCoinbaseOutputs,
	testMemWalletReorg,
	testMemWalletLockedOutputs,
}

var mainHarness *Harness

const (
	numMatureOutputs = 25
)

func TestHarness(t *testing.T) {
	// We should have (numMatureOutputs * 50 BTC) of mature unspendable
	// outputs.
	expectedBalance := dashutil.Amount(numMatureOutputs * 50 * dashutil.SatoshiPerBitcoin)
	harnessBalance := mainHarness.ConfirmedBalance()
	if harnessBalance != expectedBalance {
		t.Fatalf("expected wallet balance of %v instead have %v",
			expectedBalance, harnessBalance)
	}

	// Current tip should be at a height of numMatureOutputs plus the
	// required number of blocks for coinbase maturity.
	nodeInfo, err := mainHarness.Node.GetInfo()
	if err != nil {
		t.Fatalf("unable to execute getinfo on node: %v", err)
	}
	expectedChainHeight := numMatureOutputs + uint32(mainHarness.ActiveNet.CoinbaseMaturity)
	if uint32(nodeInfo.Blocks) != expectedChainHeight {
		t.Errorf("Chain height is %v, should be %v",
			nodeInfo.Blocks, expectedChainHeight)
	}

	for _, testCase := range harnessTestCases {
		testCase(mainHarness, t)
	}

	testTearDownAll(t)
}
