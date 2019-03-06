// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package include

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("heartbeat", "fields.yml", Asset); err != nil {
		panic(err)
	}
}

// Asset returns asset data
func Asset() string {
	return "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"
}