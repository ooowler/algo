package concurrentmap

import "fmt"

func testKey(i int) string {
	return fmt.Sprintf("k%07d", i)
}

func bucketCountFor(n int) int {
	count := n / 8
	if count < minBucketCount {
		count = minBucketCount
	}
	return nextPow2(count)
}

func makeKeys(start, count int) []string {
	keys := make([]string, count)
	for i := range keys {
		keys[i] = testKey(start + i)
	}
	return keys
}
