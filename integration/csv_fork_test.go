// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular tests due to the following build tag.
// +build rpctest

package integration

import (
	"bytes"
	"github.com/alexdcox/dashd-go/blockchain"
	"github.com/alexdcox/dashd-go/btcec"
	"github.com/alexdcox/dashd-go/chaincfg/chainhash"
	"github.com/alexdcox/dashd-go/integration/rpctest"
	"github.com/alexdcox/dashd-go/txscript"
	"github.com/alexdcox/dashd-go/wire"
	"github.com/alexdcox/dashutil"
	"runtime"
	"testing"
)

const (
	csvKey = "csv"
)

// makeTestOutput creates an on-chain output paying to a freshly generated
// p2pkh output with the specified amount.
func makeTestOutput(r *rpctest.Harness, t *testing.T,
	amt dashutil.Amount) (*btcec.PrivateKey, *wire.OutPoint, []byte, error) {

	// Create a fresh key, then send some coins to an address spendable by
	// that key.
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, nil, err
	}

	// Using the key created above, generate a pkScript which it's able to
	// spend.
	a, err := dashutil.NewAddressPubKey(key.PubKey().SerializeCompressed(), r.ActiveNet)
	if err != nil {
		return nil, nil, nil, err
	}
	selfAddrScript, err := txscript.PayToAddrScript(a.AddressPubKeyHash())
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{PkScript: selfAddrScript, Value: 1e8}

	// Next, create and broadcast a transaction paying to the output.
	fundTx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		return nil, nil, nil, err
	}
	txHash, err := r.Node.SendRawTransaction(fundTx, true)
	if err != nil {
		return nil, nil, nil, err
	}

	// The transaction created above should be included within the next
	// generated block.
	blockHash, err := r.Node.Generate(1)
	if err != nil {
		return nil, nil, nil, err
	}
	assertTxInBlock(r, t, blockHash[0], txHash)

	// Locate the output index of the coins spendable by the key we
	// generated above, this is needed in order to create a proper utxo for
	// this output.
	var outputIndex uint32
	if bytes.Equal(fundTx.TxOut[0].PkScript, selfAddrScript) {
		outputIndex = 0
	} else {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  fundTx.TxHash(),
		Index: outputIndex,
	}

	return key, utxo, selfAddrScript, nil
}

// createCSVOutput creates an output paying to a trivially redeemable CSV
// pkScript with the specified time-lock.
func createCSVOutput(r *rpctest.Harness, t *testing.T,
	numSatoshis dashutil.Amount, timeLock int32,
	isSeconds bool) ([]byte, *wire.OutPoint, *wire.MsgTx, error) {

	// Convert the time-lock to the proper sequence lock based according to
	// if the lock is seconds or time based.
	sequenceLock := blockchain.LockTimeToSequence(isSeconds,
		uint32(timeLock))

	// Our CSV script is simply: <sequenceLock> OP_CSV OP_DROP
	b := txscript.NewScriptBuilder().
		AddInt64(int64(sequenceLock)).
		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP)
	csvScript, err := b.Script()
	if err != nil {
		return nil, nil, nil, err
	}

	// Using the script generated above, create a P2SH output which will be
	// accepted into the mempool.
	p2shAddr, err := dashutil.NewAddressScriptHash(csvScript, r.ActiveNet)
	if err != nil {
		return nil, nil, nil, err
	}
	p2shScript, err := txscript.PayToAddrScript(p2shAddr)
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{
		PkScript: p2shScript,
		Value:    int64(numSatoshis),
	}

	// Finally create a valid transaction which creates the output crafted
	// above.
	tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputIndex uint32
	if !bytes.Equal(tx.TxOut[0].PkScript, p2shScript) {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  tx.TxHash(),
		Index: outputIndex,
	}

	return csvScript, utxo, tx, nil
}

// spendCSVOutput spends an output previously created by the createCSVOutput
// function. The sigScript is a trivial push of OP_TRUE followed by the
// redeemScript to pass P2SH evaluation.
func spendCSVOutput(redeemScript []byte, csvUTXO *wire.OutPoint,
	sequence uint32, targetOutput *wire.TxOut,
	txVersion int32) (*wire.MsgTx, error) {

	tx := wire.NewMsgTx(txVersion)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *csvUTXO,
		Sequence:         sequence,
	})
	tx.AddTxOut(targetOutput)

	b := txscript.NewScriptBuilder().
		AddOp(txscript.OP_TRUE).
		AddData(redeemScript)

	sigScript, err := b.Script()
	if err != nil {
		return nil, err
	}
	tx.TxIn[0].SignatureScript = sigScript

	return tx, nil
}

// assertTxInBlock asserts a transaction with the specified txid is found
// within the block with the passed block hash.
func assertTxInBlock(r *rpctest.Harness, t *testing.T, blockHash *chainhash.Hash,
	txid *chainhash.Hash) {

	block, err := r.Node.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	if len(block.Transactions) < 2 {
		t.Fatal("target transaction was not mined")
	}

	for _, txn := range block.Transactions {
		txHash := txn.TxHash()
		if txn.TxHash() == txHash {
			return
		}
	}

	_, _, line, _ := runtime.Caller(1)
	t.Fatalf("assertion failed at line %v: txid %v was not found in "+
		"block %v", line, txid, blockHash)
}