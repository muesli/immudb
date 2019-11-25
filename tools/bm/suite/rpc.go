/*
Copyright 2019 vChain, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package suite

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/codenotary/immudb/pkg/bm"
	"github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/server"
)

const Iterations = 500_000
const BatchSize = 100

var tmpDir, _ = ioutil.TempDir("", "immudb")
var immuServer = server.DefaultServer().
	WithOptions(
		server.DefaultOptions().WithDir(tmpDir),
	)
var immuClient = client.DefaultClient()

var RpcBenchmarks = []bm.Bm{
	makeRpcBenchmark("sequential write", Concurrency, Iterations, sequentialSet),
	makeRpcBenchmark("batch write", Concurrency, Iterations, batchSet),
	makeRpcBenchmark("batch write no concurrency", 1, Iterations, batchSet),
}

func sequentialSet(bm *bm.Bm, start int, end int) {
	for i := start; i < end; i++ {
		key := []byte(strconv.FormatUint(uint64(i), 10))
		_, err := immuClient.Set(bytes.NewReader(key), bytes.NewReader(V))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func batchSet(bm *bm.Bm, start int, end int) {
	var keyReaders []io.Reader
	var valueReaders []io.Reader
	for i := start; i < end; i++ {
		key := []byte(strconv.FormatUint(uint64(i), 10))
		keyReaders = append(keyReaders, bytes.NewReader(key))
		valueReaders = append(valueReaders, bytes.NewReader(V))
		if i%BatchSize == 0 || i == end-1 {
			if err := immuClient.SetBatch(&client.BatchRequest{
				Keys:   keyReaders,
				Values: valueReaders,
			}); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			keyReaders = nil
			valueReaders = nil
		}
	}
}

func makeRpcBenchmark(name string, concurrency int, iterations int,
	work func(bm *bm.Bm, start int, end int)) bm.Bm {
	return bm.Bm{
		CreateTopic: false,
		Name:        name,
		Concurrency: concurrency,
		Iterations:  iterations,
		Before: func(bm *bm.Bm) {
			go func() {
				if err := immuServer.Start(); err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}()
			if err := immuClient.Connect(); err != nil {
				_, err = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
		After: func(bm *bm.Bm) {
			if err := immuClient.Disconnect(); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := immuServer.Stop(); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := os.RemoveAll(tmpDir); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
		Work: work,
	}
}