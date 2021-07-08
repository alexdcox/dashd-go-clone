// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular tests due to the following build tag.
// +build rpctest

package integration

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/alexdcox/dashd-go/integration/rpctest"
)

// TODO: Fix broken tests (should be standalone, can start a node as part of the tests if required)
// func testGetBestBlock(r *rpctest.Harness, t *testing.T) {
// 	_, prevbestHeight, err := r.Node.GetBestBlock()
// 	if err != nil {
// 		t.Fatalf("Call to `getbestblock` failed: %v", err)
// 	}
//
// 	// Create a new block connecting to the current tip.
// 	generatedBlockHashes, err := r.Node.Generate(1)
// 	if err != nil {
// 		t.Fatalf("Unable to generate block: %v", err)
// 	}
//
// 	bestHash, bestHeight, err := r.Node.GetBestBlock()
// 	if err != nil {
// 		t.Fatalf("Call to `getbestblock` failed: %v", err)
// 	}
//
// 	// Hash should be the same as the newly submitted block.
// 	if !bytes.Equal(bestHash[:], generatedBlockHashes[0][:]) {
// 		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
// 			"hash %v", bestHash, generatedBlockHashes[0][:])
// 	}
//
// 	// Block height should now reflect newest height.
// 	if bestHeight != prevbestHeight+1 {
// 		t.Fatalf("Block heights do not match. Got %v, wanted %v",
// 			bestHeight, prevbestHeight+1)
// 	}
// }

// TODO: Fix broken tests (should be standalone, can start a node as part of the tests if required)
// func testGetBlockCount(r *rpctest.Harness, t *testing.T) {
// 	// Save the current count.
// 	currentCount, err := r.Node.GetBlockCount()
// 	if err != nil {
// 		t.Fatalf("Unable to get block count: %v", err)
// 	}
//
// 	if _, err := r.Node.Generate(1); err != nil {
// 		t.Fatalf("Unable to generate block: %v", err)
// 	}
//
// 	// Count should have increased by one.
// 	newCount, err := r.Node.GetBlockCount()
// 	if err != nil {
// 		t.Fatalf("Unable to get block count: %v", err)
// 	}
// 	if newCount != currentCount+1 {
// 		t.Fatalf("Block count incorrect. Got %v should be %v",
// 			newCount, currentCount+1)
// 	}
// }

// TODO: Fix broken tests (should be standalone, can start a node as part of the tests if required)
// func testGetBlockHash(r *rpctest.Harness, t *testing.T) {
// 	// Create a new block connecting to the current tip.
// 	generatedBlockHashes, err := r.Node.Generate(1)
// 	if err != nil {
// 		t.Fatalf("Unable to generate block: %v", err)
// 	}
//
// 	info, err := r.Node.GetInfo()
// 	if err != nil {
// 		t.Fatalf("call to getinfo cailed: %v", err)
// 	}
//
// 	blockHash, err := r.Node.GetBlockHash(int64(info.Blocks))
// 	if err != nil {
// 		t.Fatalf("Call to `getblockhash` failed: %v", err)
// 	}
//
// 	// Block hashes should match newly created block.
// 	if !bytes.Equal(generatedBlockHashes[0][:], blockHash[:]) {
// 		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
// 			"hash %v", blockHash, generatedBlockHashes[0][:])
// 	}
// }

var rpcTestCases = []rpctest.HarnessTestCase{
	testGetBestBlock,
	testGetBlockCount,
	testGetBlockHash,
}

var primaryHarness *rpctest.Harness

func TestRpcServer(t *testing.T) {
	var currentTestNum int
	defer func() {
		// If one of the integration tests caused a panic within the main
		// goroutine, then tear down all the harnesses in order to avoid
		// any leaked btcd processes.
		if r := recover(); r != nil {
			fmt.Println("recovering from test panic: ", r)
			if err := rpctest.TearDownAll(); err != nil {
				fmt.Println("unable to tear down all harnesses: ", err)
			}
			t.Fatalf("test #%v panicked: %s", currentTestNum, debug.Stack())
		}
	}()

	for _, testCase := range rpcTestCases {
		testCase(primaryHarness, t)

		currentTestNum++
	}
}
