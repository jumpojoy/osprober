// Copyright 2023 The Osprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package formated_file;

import (
	"sync"
	"time"
	"context"
	"encoding/json"
	"io/ioutil"
        "os"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/jumpojoy/osprober/surfacers/formated_file/proto"

)

type FileMapEvent struct {
	Name string
	Timestamp time.Time
        Labels map[string]string
	Metrics map[string]string
}

type FileMapSurfacer struct {
    mu     sync.RWMutex
    Received map[string]*FileMapEvent
    l *logger.Logger
}

const metricWriteTime = 10 * time.Second

const metricExpirationTime = 6 * metricWriteTime

func New(config *configpb.SurfacerConf, l *logger.Logger) (*FileMapSurfacer, error) {
    fms := &FileMapSurfacer{
	    Received: make(map[string] *FileMapEvent),
	    l: l,
    }
    go func() {
	dst := config.GetFilePath()
	for true {
            time.Sleep(metricWriteTime)
	    staleTime := time.Now().Truncate(metricExpirationTime)
	    var expiredMetrics []string
	    fms.mu.Lock()
	    for name, metric := range fms.Received {
                    if metric.Timestamp.Before(staleTime) {
                                expiredMetrics = append(expiredMetrics, name)
                                delete(fms.Received, name)
                        }

            }
	    jsonString, _ := json.Marshal(fms.Received)
	    ioutil.WriteFile(dst, jsonString, os.ModePerm)
	    fms.mu.Unlock()
	    l.Debugf("Dump metrics to %s", dst)
	    if len(expiredMetrics) > 0 {
		    l.Debugf("Drop metrics %v", expiredMetrics)
            }
    	}
    }()

    return fms, nil
}


func (ts *FileMapSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	dst := em.Label("dst")
	if dst != "" {
		labels := make(map[string]string)
		for _, k := range em.LabelsKeys() {
		        labels[k] = em.Label(k)
	        }
                metrics := make(map[string]string)
                for _, mk := range em.MetricsKeys() {
		        metrics[mk] = em.Metric(mk).String()
	        }

		ts.mu.Lock()
		ts.Received[dst] = &FileMapEvent{Name: dst, Timestamp: em.Timestamp, Labels: labels, Metrics: metrics}
		ts.mu.Unlock()
	}
}
