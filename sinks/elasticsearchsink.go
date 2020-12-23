/*
Copyright 2017 Heptio Inc.

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

package sinks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// GlogSink is the most basic sink
// Useful when you already have ELK/EFK Stack
type ElasticSearchSink struct {
	// TODO: create a channel and buffer for scaling
	idx esutil.BulkIndexer
}

// NewGlogSink will create a new
func NewElasticSearchSink() EventSinkInterface {

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://10.0.10.34:9200",
			"http://10.0.10.35:9200",
			"http://10.0.10.43:9200",
			"http://10.0.20.30:9200",
			"http://10.0.20.31:9200",
		},
		RetryOnStatus: []int{502, 503, 504, 429}, // Add 429 to the list of retryable statuses
		RetryBackoff:  func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
		MaxRetries:    5,
		EnableMetrics: true,
	})

	idx, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:      url.QueryEscape("<k8s_event_log_{now/M{YYYY.MM}}>"),
		Client:     es,
		NumWorkers: 0,
		FlushBytes: 0,
	})
	if err != nil {
		panic(err)
	}


	return &ElasticSearchSink{
		idx: idx,
	}
}

// UpdateEvents implements the EventSinkInterface
func (gs *ElasticSearchSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	eData := NewEventData(eNew, eOld)

	if eJSONBytes, err := json.Marshal(eData); err == nil {
		ctx := context.Background()
		if err := gs.idx.Add(ctx,
			esutil.BulkIndexerItem{
				Action: "create",
				Body:   bytes.NewReader(eJSONBytes),
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					//log.Printf("Indexed %s/%s", res.Index, res.DocumentID)
				},
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						glog.Warningf("OnFailure: %v", err)
					} else {
						if res.Error.Type != "" {
							glog.Warningf("%s:%s", res.Error.Type, res.Error.Reason)
						} else {
							glog.Warningf("%s/%s %s (%d)", res.Index, res.DocumentID, res.Result, res.Status)
						}

					}
				},
			}); err != nil {
			glog.Warningf("indexer: %s", err)
		}
	} else {
		glog.Warningf("Failed to json serialize event: %v", err)
	}
}
