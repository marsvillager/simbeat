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

package graphite

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "graphite", Asset); err != nil {
		panic(err)
	}
}

// Asset returns asset data
func Asset() string {
	return "eJx8jz0SwiAUhHtOsZM+F6Cwc6w8BMoamZDAAFFzeyc/ZkKMvvJ7vG+XEjV7iSoofzeJAkgmWUoUpxkVAtCM12B8Mq6VOAgAywXOTnd2OAy0VJESFyYlgJuh1VGOr0u0qmGWM0zq/Qhd52eyk5Sb1rbI8GBY8J7vp3OaL8PyiUqt6LbAugRfqvGW2e7TpGb/dEFvdn/6DHOchFOoeAcAAP//RydvYQ=="
}
