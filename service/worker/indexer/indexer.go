// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package indexer

import (
	"fmt"
	"github.com/olivere/elastic"
	"github.com/uber-common/bark"
	"github.com/uber/cadence/common/logging"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/service/dynamicconfig"
)

type (
	// Indexer used to consumer data from kafka then send to ElasticSearch
	Indexer struct {
		config              *Config
		kafkaClient         messaging.Client
		esClient            *elastic.Client
		logger              bark.Logger
		metricsClient       metrics.Client
		visibilityProcessor *indexProcessor
	}

	// Config contains all configs for indexer
	Config struct {
		EnableIndexer            dynamicconfig.BoolPropertyFn
		IndexerConcurrency       dynamicconfig.IntPropertyFn
		ESProcessorNumOfWorkers  dynamicconfig.IntPropertyFn
		ESProcessorBulkActions   dynamicconfig.IntPropertyFn // max number of requests in bulk
		ESProcessorBulkSize      dynamicconfig.IntPropertyFn // max total size of bytes in bulk
		ESProcessorFlushInterval dynamicconfig.DurationPropertyFn
		ESProcessorRetryInterval dynamicconfig.DurationPropertyFn // retry interval on ES catastrophe
	}
)

const (
	visibilityProcessorName = "visibility-processor"
)

// NewIndexer create a new Indexer
func NewIndexer(config *Config, client messaging.Client, esClient *elastic.Client, logger bark.Logger, metricsClient metrics.Client) *Indexer {
	logger = logger.WithFields(bark.Fields{
		logging.TagWorkflowComponent: logging.TagValueIndexerComponent,
	})
	return &Indexer{
		config:        config,
		kafkaClient:   client,
		esClient:      esClient,
		logger:        logger,
		metricsClient: metricsClient,
	}
}

// Start indexer
func (x Indexer) Start() error {
	visibilityApp := messaging.VisibilityAppName
	visConsumerName := getConsumerName(visibilityApp)
	x.visibilityProcessor = newIndexProcessor(visibilityApp, visConsumerName, x.kafkaClient, x.esClient, visibilityProcessorName, x.config, x.logger, x.metricsClient)
	return x.visibilityProcessor.Start()
}

// Stop indexer
func (x Indexer) Stop() {
	x.visibilityProcessor.Stop()
}

func getConsumerName(topic string) string {
	return fmt.Sprintf("%s-consumer", topic)
}
