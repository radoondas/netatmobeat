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

package redis

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "redis", Asset); err != nil {
		panic(err)
	}
}

// Asset returns asset data
func Asset() string {
	return "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"
}