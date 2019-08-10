// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package netflow

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("filebeat", "netflow", asset.ModuleFieldsPri, AssetNetflow); err != nil {
		panic(err)
	}
}

// AssetNetflow returns asset data.
// This is the base64 encoded gzipped contents of module/netflow.
func AssetNetflow() string {
	return "eJw8jjFOw0AQRfs9xbtAcoAtqFCkFKAUINGazBiPWHas3Ymt3B4Z4fTv/f8OfOs9UzXG4uvhx+VWNEFYFM28apyKrwlE+7XZHOY185QAXv5gRm80vaotVr92g6EK58vp/ME2vAHepOOLNt6fL0feJuVxB+LaqR4MIoymRTqfevcqrNMQxKR7JVbnWzA3X0y0HxP/Qk6/AQAA//9CcUYh"
}