//
// Copyright (C) 2021 IBM Corporation.
//
// Authors:
// Andreas Schade <san@zurich.ibm.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package main

import (
	"encoding/csv"
	"os"
	"sync"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-apis/go/sfgo"

	"github.com/sysflow-telemetry/sf-processor/core/policyengine/engine"
)

var Action HashAction
var instance map[string]string
var once sync.Once

type HashAction struct{}

func (a *HashAction) GetName() string {
	return "hash"
}

func (a *HashAction) GetFunc() engine.ActionFunc {
	return process
}

func process(r *engine.Record) error {
	m := getHashMap()
	sfType := engine.Mapper.MapStr(engine.SF_TYPE)(r)
	if sfType == sfgo.TyFFStr {
		fpath := engine.Mapper.MapStr(engine.SF_FILE_PATH)(r)
		if h, ok := m[fpath]; ok {
			r.Ctx.SetHashes(engine.HASH_TYPE_FILE, &engine.HashSet{Md5: h})
			r.Ctx.AddTag("file_md5_hash:" + h)
		}
	}
	fpath := engine.Mapper.MapStr(engine.SF_PROC_EXE)(r)
	if h, ok := m[fpath]; ok {
		r.Ctx.SetHashes(engine.HASH_TYPE_PROC, &engine.HashSet{Md5: h})
		r.Ctx.AddTag("process_md5_hash:" + h)
	}
	return nil
}

func getHashMap() map[string]string {
	once.Do(func() {
		instance = createHashMap()
	})
	return instance
}

func createHashMap() map[string]string {
	data := readCSV()
	m := make(map[string]string)
	for _, line := range data {
		if len(line) == 2 {
			m[line[1]] = line[0]
		}
	}
	return m
}

func readCSV() (data [][]string) {
	f, err := os.Open("../plugins/actions/hash/hashes.csv")
	if err != nil {
		logger.Error.Println("Unable to open csv file", err)
		return
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	data, err = csvReader.ReadAll()
	if err != nil {
		logger.Error.Println("Unable to read csv file", err)
	}
	return
}

// This function is not run when module is used as a plugin.
func main() {}
