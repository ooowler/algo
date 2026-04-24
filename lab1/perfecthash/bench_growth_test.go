package perfecthash

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
)

func growthPHSizes() []int {
	return []int{4096, 8192, 16384, 32768, 65536, 131072, 262144}
}

func makePHInput(n int) ([]string, []string) {
	keys := make([]string, n)
	values := make([]string, n)
	keyWidth := len(strconv.Itoa(growthPHSizes()[len(growthPHSizes())-1] - 1))
	valueWidth := len(strconv.Itoa(growthPHSizes()[len(growthPHSizes())-1] * 2))
	for i := range keys {
		keys[i] = intKeyDec(i, keyWidth)
		values[i] = intKeyDec(i*2, valueWidth)
	}
	return keys, values
}

func intKeyDec(x, width int) string {
	if width < 1 {
		width = 1
	}
	return fmt.Sprintf("%0*d", width, x)
}

func BenchmarkGrowthPH_Build(b *testing.B) {
	for _, n := range growthPHSizes() {
		keys, values := makePHInput(n)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Build(keys, values)
			}
		})
	}
}

func BenchmarkGrowthPH_Get(b *testing.B) {
	for _, n := range growthPHSizes() {
		keys, values := makePHInput(n)
		index := Build(keys, values)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			warm := n * 4
			if warm > 200000 {
				warm = 200000
			}
			for i := 0; i < warm; i++ {
				index.Get(keys[i%n])
			}
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				index.Get(keys[i%n])
			}
		})
	}
}
