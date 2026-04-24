package lsh

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type lshMetric struct {
	N                  int     `json:"n"`
	Seeds              int     `json:"seeds"`
	TruePairsMean      float64 `json:"true_pairs_mean"`
	TruePairsStd       float64 `json:"true_pairs_std"`
	CandidateCountMean float64 `json:"candidate_count_mean"`
	CandidateCountStd  float64 `json:"candidate_count_std"`
	CandidateRatioMean float64 `json:"candidate_ratio_mean"`
	CandidateRatioStd  float64 `json:"candidate_ratio_std"`
	RecallMean         float64 `json:"recall_mean"`
	RecallStd          float64 `json:"recall_std"`
	PrecisionMean      float64 `json:"precision_mean"`
	PrecisionStd       float64 `json:"precision_std"`
	BuildMsMean        float64 `json:"build_ms_mean"`
	BuildMsStd         float64 `json:"build_ms_std"`
	AddNsMean          float64 `json:"add_ns_mean"`
	AddNsStd           float64 `json:"add_ns_std"`
	FindMsMean         float64 `json:"find_ms_mean"`
	FindMsStd          float64 `json:"find_ms_std"`
	NaiveMsMean        float64 `json:"naive_ms_mean"`
	NaiveMsStd         float64 `json:"naive_ms_std"`
	SpeedupMean        float64 `json:"speedup_mean"`
	SpeedupStd         float64 `json:"speedup_std"`
}

func TestLSH_Metrics(t *testing.T) {
	if os.Getenv("LSH_METRICS") == "" {
		t.Skip("set LSH_METRICS=1 to emit lsh_metrics.json")
	}
	const threshold = 1.5
	sizes := growthDatasetSizes()
	seeds := []int64{11, 17, 23, 29, 35}
	out := make([]lshMetric, 0, len(sizes))

	for _, n := range sizes {
		truePairsSamples := make([]float64, 0, len(seeds))
		candidateCountSamples := make([]float64, 0, len(seeds))
		candidateRatioSamples := make([]float64, 0, len(seeds))
		recallSamples := make([]float64, 0, len(seeds))
		precisionSamples := make([]float64, 0, len(seeds))
		buildSamples := make([]float64, 0, len(seeds))
		addSamples := make([]float64, 0, len(seeds))
		findSamples := make([]float64, 0, len(seeds))
		naiveSamples := make([]float64, 0, len(seeds))
		speedupSamples := make([]float64, 0, len(seeds))

		for _, seed := range seeds {
			points := GenerateDataset(n, threshold, seed)

			runtime.GC()
			buildStart := time.Now()
			_ = BuildIndex(10, 8, threshold, rand.New(rand.NewSource(seed+1000)), points)
			buildSamples = append(buildSamples, float64(time.Since(buildStart).Microseconds())/1000.0)

			addBatch := n / 10
			if addBatch < 128 {
				addBatch = 128
			}
			addPoints := GenerateDataset(addBatch, threshold, seed+2000)
			addIndex := BuildIndex(10, 8, threshold, rand.New(rand.NewSource(seed+3000)), points)
			runtime.GC()
			addStart := time.Now()
			for _, point := range addPoints {
				addIndex.Add(point)
			}
			addDuration := time.Since(addStart).Nanoseconds()
			addSamples = append(addSamples, float64(addDuration)/float64(addBatch))

			findIndex := BuildIndex(10, 8, threshold, rand.New(rand.NewSource(seed+4000)), points)
			runtime.GC()
			findStart := time.Now()
			foundPairs, stats := findIndex.FindDuplicatesWithStats()
			findMs := float64(time.Since(findStart).Microseconds()) / 1000.0
			findSamples = append(findSamples, findMs)

			runtime.GC()
			naiveStart := time.Now()
			naivePairs := NaiveFindDuplicates(points, threshold)
			naiveMs := float64(time.Since(naiveStart).Microseconds()) / 1000.0
			naiveSamples = append(naiveSamples, naiveMs)

			trueSet := pairSetInt(naivePairs)
			foundSet := pairSetInt(foundPairs)
			truePositives := 0
			for pair := range foundSet {
				if trueSet[pair] {
					truePositives++
				}
			}
			recall := 1.0
			if len(trueSet) > 0 {
				recall = float64(truePositives) / float64(len(trueSet))
			}
			precision := 1.0
			if len(foundSet) > 0 {
				precision = float64(truePositives) / float64(len(foundSet))
			}
			allPairs := int64(len(points)) * int64(len(points)-1) / 2
			candidateRatio := 0.0
			if allPairs > 0 {
				candidateRatio = float64(stats.UniqueCandidates) / float64(allPairs)
			}

			truePairsSamples = append(truePairsSamples, float64(len(trueSet)))
			candidateCountSamples = append(candidateCountSamples, float64(stats.UniqueCandidates))
			candidateRatioSamples = append(candidateRatioSamples, candidateRatio)
			recallSamples = append(recallSamples, recall)
			precisionSamples = append(precisionSamples, precision)
			if findMs > 0 {
				speedupSamples = append(speedupSamples, naiveMs/findMs)
			} else {
				speedupSamples = append(speedupSamples, 0)
			}
		}

		truePairsMean, truePairsStd := meanStd(truePairsSamples)
		candidateCountMean, candidateCountStd := meanStd(candidateCountSamples)
		candidateRatioMean, candidateRatioStd := meanStd(candidateRatioSamples)
		recallMean, recallStd := meanStd(recallSamples)
		precisionMean, precisionStd := meanStd(precisionSamples)
		buildMean, buildStd := meanStd(buildSamples)
		addMean, addStd := meanStd(addSamples)
		findMean, findStd := meanStd(findSamples)
		naiveMean, naiveStd := meanStd(naiveSamples)
		speedupMean, speedupStd := meanStd(speedupSamples)

		out = append(out, lshMetric{
			N:                  n,
			Seeds:              len(seeds),
			TruePairsMean:      truePairsMean,
			TruePairsStd:       truePairsStd,
			CandidateCountMean: candidateCountMean,
			CandidateCountStd:  candidateCountStd,
			CandidateRatioMean: candidateRatioMean,
			CandidateRatioStd:  candidateRatioStd,
			RecallMean:         recallMean,
			RecallStd:          recallStd,
			PrecisionMean:      precisionMean,
			PrecisionStd:       precisionStd,
			BuildMsMean:        buildMean,
			BuildMsStd:         buildStd,
			AddNsMean:          addMean,
			AddNsStd:           addStd,
			FindMsMean:         findMean,
			FindMsStd:          findStd,
			NaiveMsMean:        naiveMean,
			NaiveMsStd:         naiveStd,
			SpeedupMean:        speedupMean,
			SpeedupStd:         speedupStd,
		})
	}

	dir := filepath.Join("..", "results")
	_ = os.MkdirAll(dir, 0755)
	file, err := os.Create(filepath.Join(dir, "lsh_metrics.json"))
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

func pairSetInt(pairs [][2]int) map[[2]int]bool {
	set := make(map[[2]int]bool, len(pairs))
	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		if a > b {
			a, b = b, a
		}
		set[[2]int{a, b}] = true
	}
	return set
}
