// Copyright 2019 ScyllaDB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"github.com/scylladb/gemini/pkg/generators"
	"github.com/scylladb/gemini/pkg/typedef"

	"go.uber.org/zap"
)

func createGenerators(
	schema *typedef.Schema,
	schemaConfig typedef.SchemaConfig,
	seed, distributionSize uint64,
	logger *zap.Logger,
) (generators.Generators, error) {
	partitionRangeConfig := schemaConfig.GetPartitionRangeConfig()

	var gs []*generators.Generator
	for id := range schema.Tables {
		table := schema.Tables[id]

		distFunc, err := createDistributionFunc(partitionKeyDistribution, partitionCount, seed, stdDistMean, oneStdDev)
		if err != nil {
			return nil, err
		}

		gCfg := &generators.Config{
			PartitionsRangeConfig:      partitionRangeConfig,
			PartitionsCount:            distributionSize,
			PartitionsDistributionFunc: distFunc,
			Seed:                       seed,
			PkUsedBufferSize:           pkBufferReuseSize,
		}
		g := generators.NewGenerator(table, gCfg, logger.Named("generators"))
		gs = append(gs, g)
	}
	return gs, nil
}
