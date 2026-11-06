// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/google/blueprint"
	"github.com/google/blueprint/metrics"
	"github.com/google/blueprint/proptools"

	"google.golang.org/protobuf/proto"

	soong_metrics_proto "android/soong/ui/metrics/metrics_proto"
)

var soongMetricsOnceKey = NewOnceKey("soong metrics")

type soongMetrics struct {
	modules               int
	variants              int
	incrementalModules    incrementalMetrics
	incrementalSingletons incrementalMetrics
	perfCollector         perfCollector
	incrementalAnalysis   bool
	incrementalEnabled    bool
}

type incrementalMetrics struct {
	supported              int
	cacheHit               int
	providersCached        map[string]int
	providersRestored      map[string]int
	totalProvidersCached   int
	totalProvidersRestored int
}

type perfCollector struct {
	events []*soong_metrics_proto.PerfCounters
	stop   chan<- bool
}

func getSoongMetrics(config Config) *soongMetrics {
	return config.Once(soongMetricsOnceKey, func() interface{} {
		return &soongMetrics{
			incrementalModules: incrementalMetrics{
				providersCached:   make(map[string]int),
				providersRestored: make(map[string]int),
			},
		}
	}).(*soongMetrics)
}

func soongMetricsSingletonFactory() Singleton { return soongMetricsSingleton{} }

type soongMetricsSingleton struct{}

func (soongMetricsSingleton) IncrementalSupported() bool {
	// always run this to collect metrics.
	return false
}

func (soongMetricsSingleton) GenerateBuildActions(ctx SingletonContext) {
	metrics := getSoongMetrics(ctx.Config())
	ctx.VisitAllModuleProxies(func(m ModuleProxy) {
		if ctx.PrimaryModuleProxy(m) == m {
			metrics.modules++
		}
		info := m.IncrementalInfo()
		if info.IncrementalSupported {
			metrics.incrementalModules.supported++
		}
		if info.IncrementalRestored {
			metrics.incrementalModules.cacheHit++

			for i, unRestored := range info.HasUnrestoredProvider {
				if info.ProviderInitialValueHashes[i] != proptools.ZeroHash && blueprint.ProviderMutator(i) == "" {
					metrics.incrementalModules.providersCached[blueprint.ProviderType(i)]++
					metrics.incrementalModules.totalProvidersCached++
					if !unRestored {
						metrics.incrementalModules.providersRestored[blueprint.ProviderType(i)]++
						metrics.incrementalModules.totalProvidersRestored++
					}
				}
			}
		}
		metrics.variants++
	})

	ctx.VisitAllSingletons(func(s blueprint.SingletonProxy) {
		info := s.IncrementalInfo()
		if info.IncrementalSupported {
			metrics.incrementalSingletons.supported++
		}
		if info.IncrementalRestored {
			metrics.incrementalSingletons.cacheHit++
		}
	})

	metrics.incrementalAnalysis = ctx.GetIncrementalAnalysis()
	metrics.incrementalEnabled = ctx.GetIncrementalEnabled()
}

func collectMetrics(ctx *Context, config Config, eventHandler *metrics.EventHandler) *soong_metrics_proto.SoongBuildMetrics {
	metrics := &soong_metrics_proto.SoongBuildMetrics{
		IncrementalInfo: &soong_metrics_proto.IncrementalInfo{
			ModuleMetrics: &soong_metrics_proto.IncrementalMetrics{
				Providers: make(map[string]*soong_metrics_proto.ProviderMetrics),
			},
			SingletonMetrics: &soong_metrics_proto.IncrementalMetrics{},
		},
	}

	soongMetrics := getSoongMetrics(config)
	if soongMetrics.modules > 0 {
		metrics.Modules = proto.Uint32(uint32(soongMetrics.modules))
		metrics.Variants = proto.Uint32(uint32(soongMetrics.variants))
	}
	metrics.IncrementalInfo.IncrementalAnalysisUsed = proto.Bool(soongMetrics.incrementalAnalysis)
	metrics.IncrementalInfo.IncrementalAnalysisEnabled = proto.Bool(soongMetrics.incrementalEnabled)
	metrics.IncrementalInfo.ModuleMetrics.Supported = proto.Uint32(uint32(soongMetrics.incrementalModules.supported))
	metrics.IncrementalInfo.ModuleMetrics.CacheHit = proto.Uint32(uint32(soongMetrics.incrementalModules.cacheHit))
	for k, v := range soongMetrics.incrementalModules.providersCached {
		metrics.IncrementalInfo.ModuleMetrics.Providers[k] = &soong_metrics_proto.ProviderMetrics{
			ProvidersCached: proto.Uint32(uint32(v)),
		}
	}
	for k, v := range soongMetrics.incrementalModules.providersRestored {
		metrics.IncrementalInfo.ModuleMetrics.Providers[k].ProvidersRestored = proto.Uint32(uint32(v))
	}
	metrics.IncrementalInfo.ModuleMetrics.TotalProvidersCached = proto.Uint32(uint32(soongMetrics.incrementalModules.totalProvidersCached))
	metrics.IncrementalInfo.ModuleMetrics.TotalProvidersRestored = proto.Uint32(uint32(soongMetrics.incrementalModules.totalProvidersRestored))

	metrics.IncrementalInfo.SingletonMetrics.Supported = proto.Uint32(uint32(soongMetrics.incrementalSingletons.supported))
	metrics.IncrementalInfo.SingletonMetrics.CacheHit = proto.Uint32(uint32(soongMetrics.incrementalSingletons.cacheHit))

	restoredGlobMetrics := ctx.RestoredGlobMetrics()
	metrics.IncrementalInfo.CachedGlobs = proto.Uint64(restoredGlobMetrics.CachedGlobs)
	metrics.IncrementalInfo.RestoredGlobs = proto.Uint64(restoredGlobMetrics.RestoredGlobs)

	if config.IsActionSandboxedBuild() && config.ActionSandboxMetrics() != nil {
		sandboxMetrics := config.ActionSandboxMetrics()
		metrics.SandboxMetrics = &soong_metrics_proto.SandboxMetrics{
			TotalRules:    proto.Uint32(uint32(sandboxMetrics.TotalRules())),
			DisabledRules: proto.Uint32(uint32(sandboxMetrics.DisabledRules())),
		}
		for _, rule := range sandboxMetrics.Actions() {
			metrics.SandboxMetrics.Actions = append(metrics.SandboxMetrics.Actions, &soong_metrics_proto.SandboxAction{
				Name:      &rule.Name,
				Sandboxed: &rule.Sandboxed,
			})
		}
	}

	if soongMetrics.incrementalAnalysis {
		incrementalDbSize := getIncrementalInfoDbSize(config)
		metrics.IncrementalInfo.IncrementalDbSize = &incrementalDbSize
	}

	soongMetrics.perfCollector.stop <- true
	metrics.PerfCounters = soongMetrics.perfCollector.events

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	metrics.MaxHeapSize = proto.Uint64(memStats.HeapSys)
	metrics.TotalAllocCount = proto.Uint64(memStats.Mallocs)
	metrics.TotalAllocSize = proto.Uint64(memStats.TotalAlloc)

	for _, event := range eventHandler.CompletedEvents() {
		perfInfo := soong_metrics_proto.PerfInfo{
			Description: proto.String(event.Id),
			Name:        proto.String("soong_build"),
			StartTime:   proto.Uint64(uint64(event.Start.UnixNano())),
			RealTime:    proto.Uint64(event.RuntimeNanoseconds()),
		}
		metrics.Events = append(metrics.Events, &perfInfo)
	}

	return metrics
}

func getIncrementalInfoDbSize(config Config) int64 {
	addFileSizeInDir := func(dirPath string) int64 {
		var ret int64 = 0
		filepath.WalkDir(absolutePath(dirPath), func(path string, f fs.DirEntry, err error) error {
			if err != nil {
				panic(fmt.Errorf("Could not walk directory %s due to err %s\n", dirPath, err))
			}
			if f.IsDir() {
				return nil
			}
			fileInfo, err := os.Stat(path)
			if err != nil {
				panic(fmt.Errorf("Could not open %s due to %s\n", path, err))
			} else {
				ret += fileInfo.Size()
			}
			return nil
		})
		return ret
	}
	var dbFileSize int64 = 0
	for _, dir := range blueprint.IncrementalInfoDbNames {
		dbFileSize += addFileSizeInDir(filepath.Join(config.SoongOutDir(), dir))
	}
	return dbFileSize
}

func StartBackgroundMetrics(config Config) {
	perfCollector := &getSoongMetrics(config).perfCollector
	stop := make(chan bool)
	perfCollector.stop = stop

	previousTime := time.Now()
	previousCpuTime := readCpuTime()

	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-stop:
				ticker.Stop()
				return
			case <-ticker.C:
				// carry on
			}

			currentTime := time.Now()

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			currentCpuTime := readCpuTime()

			interval := currentTime.Sub(previousTime)
			intervalCpuTime := currentCpuTime - previousCpuTime
			intervalCpuPercent := intervalCpuTime * 100 / interval

			// heapAlloc is the memory that has been allocated on the heap but not yet GC'd.  It may be referenced,
			// or unrefenced but not yet GC'd.
			heapAlloc := memStats.HeapAlloc
			// heapUnused is the memory that was previously used by the heap, but is currently not used.  It does not
			// count memory that was used and then returned to the OS.
			heapUnused := memStats.HeapIdle - memStats.HeapReleased
			// heapOverhead is the memory used by the allocator and GC
			heapOverhead := memStats.MSpanSys + memStats.MCacheSys + memStats.GCSys
			// otherMem is the memory used outside of the heap.
			otherMem := memStats.Sys - memStats.HeapSys - heapOverhead

			perfCollector.events = append(perfCollector.events, &soong_metrics_proto.PerfCounters{
				Time: proto.Uint64(uint64(currentTime.UnixNano())),
				Groups: []*soong_metrics_proto.PerfCounterGroup{
					{
						Name: proto.String("cpu"),
						Counters: []*soong_metrics_proto.PerfCounter{
							{Name: proto.String("cpu_percent"), Value: proto.Int64(int64(intervalCpuPercent))},
						},
					}, {
						Name: proto.String("memory"),
						Counters: []*soong_metrics_proto.PerfCounter{
							{Name: proto.String("heap_alloc"), Value: proto.Int64(int64(heapAlloc))},
							{Name: proto.String("heap_unused"), Value: proto.Int64(int64(heapUnused))},
							{Name: proto.String("heap_overhead"), Value: proto.Int64(int64(heapOverhead))},
							{Name: proto.String("other"), Value: proto.Int64(int64(otherMem))},
						},
					},
				},
			})

			previousTime = currentTime
			previousCpuTime = currentCpuTime
		}
	}()
}

func readCpuTime() time.Duration {
	if runtime.GOOS != "linux" {
		return 0
	}

	stat, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}

	endOfComm := bytes.LastIndexByte(stat, ')')
	if endOfComm < 0 || endOfComm > len(stat)-2 {
		return 0
	}

	stat = stat[endOfComm+2:]

	statFields := bytes.Split(stat, []byte{' '})
	// This should come from sysconf(_SC_CLK_TCK), but there's no way to call that from Go.  Assume it's 100,
	// which is the value for all platforms we support.
	const HZ = 100
	const MS_PER_HZ = 1e3 / HZ * time.Millisecond

	const STAT_UTIME_FIELD = 14 - 2
	const STAT_STIME_FIELD = 15 - 2
	if len(statFields) < STAT_STIME_FIELD {
		return 0
	}
	userCpuTicks, err := strconv.ParseUint(string(statFields[STAT_UTIME_FIELD]), 10, 64)
	if err != nil {
		return 0
	}
	kernelCpuTicks, _ := strconv.ParseUint(string(statFields[STAT_STIME_FIELD]), 10, 64)
	if err != nil {
		return 0
	}
	return time.Duration(userCpuTicks+kernelCpuTicks) * MS_PER_HZ
}

func WriteMetrics(ctx *Context, config Config, eventHandler *metrics.EventHandler, metricsFile string) error {
	metrics := collectMetrics(ctx, config, eventHandler)

	buf, err := proto.Marshal(metrics)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(absolutePath(metricsFile), buf, 0666)
	if err != nil {
		return err
	}

	return nil
}
