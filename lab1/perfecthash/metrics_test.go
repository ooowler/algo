package perfecthash

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type phfMetric struct {
	N                      int     `json:"n"`
	TableSize              int     `json:"table_size"`
	LoadFactor             float64 `json:"load_factor"`
	DisplacementBitsPerKey float64 `json:"displacement_bits_per_key"`
	BuildRuns              int     `json:"build_runs"`
	BuildNsPerKeyMean      float64 `json:"build_ns_per_key_mean"`
	BuildNsPerKeyStd       float64 `json:"build_ns_per_key_std"`
	QueryRuns              int     `json:"query_runs"`
	QueryNsMean            float64 `json:"query_ns_mean"`
	QueryNsStd             float64 `json:"query_ns_std"`
}

func TestPHF_Metrics(t *testing.T) {
	if os.Getenv("PHF_METRICS") == "" {
		t.Skip("set PHF_METRICS=1 to emit phf_metrics.json")
	}
	sizes := append([]int{1024}, growthPHSizes()...)
	out := make([]phfMetric, 0, len(sizes))

	for _, n := range sizes {
		keys, values := makePHInput(n)
		buildRuns := 5
		if n >= 65536 {
			buildRuns = 3
		}
		queryRuns := 8
		if n >= 65536 {
			queryRuns = 6
		}

		buildSamples := make([]float64, 0, buildRuns)
		var index *Index
		for i := 0; i < buildRuns; i++ {
			runtime.GC()
			start := time.Now()
			index = Build(keys, values)
			buildSamples = append(buildSamples, float64(time.Since(start).Nanoseconds())/float64(n))
		}

		warm := n * 8
		if warm > 400000 {
			warm = 400000
		}
		for i := 0; i < warm; i++ {
			index.Get(keys[i%n])
		}

		querySamples := make([]float64, 0, queryRuns)
		batch := 100000
		if n >= 65536 {
			batch = 200000
		}
		for i := 0; i < queryRuns; i++ {
			runtime.GC()
			start := time.Now()
			for j := 0; j < batch; j++ {
				index.Get(keys[j%n])
			}
			querySamples = append(querySamples, float64(time.Since(start).Nanoseconds())/float64(batch))
		}

		buildMean, buildStd := meanStd(buildSamples)
		queryMean, queryStd := meanStd(querySamples)
		out = append(out, phfMetric{
			N:                      n,
			TableSize:              index.tableSize,
			LoadFactor:             index.LoadFactor(),
			DisplacementBitsPerKey: index.BitsPerKey(),
			BuildRuns:              buildRuns,
			BuildNsPerKeyMean:      buildMean,
			BuildNsPerKeyStd:       buildStd,
			QueryRuns:              queryRuns,
			QueryNsMean:            queryMean,
			QueryNsStd:             queryStd,
		})
	}

	dir := filepath.Join("..", "results")
	_ = os.MkdirAll(dir, 0755)
	file, err := os.Create(filepath.Join(dir, "phf_metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		t.Fatal(err)
	}
}

func meanStd(samples []float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, sample := range samples {
		sum += sample
	}
	mean := sum / float64(len(samples))
	if len(samples) == 1 {
		return mean, 0
	}
	ss := 0.0
	for _, sample := range samples {
		delta := sample - mean
		ss += delta * delta
	}
	return mean, math.Sqrt(ss / float64(len(samples)-1))
}
