package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type sample struct {
	n      float64
	nsOp   float64
	bytesB float64
}

var lineRe = regexp.MustCompile(
	`^(BenchmarkGrowth\S+)/N(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op`,
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		log.Fatal("запускай из корня модуля (каталог с go.mod)")
	}

	out, err := exec.Command(
		"go", "test", "./hashtable", "./perfecthash", "./lsh",
		"-run=^$", "-bench=BenchmarkGrowth", "-benchmem", "-benchtime=150ms", "-count=1", "-timeout=180s",
	).CombinedOutput()
	if err != nil {
		log.Printf("%s", out)
		log.Fatal(err)
	}

	_ = os.MkdirAll(filepath.Join(root, "results"), 0755)
	_ = os.MkdirAll(filepath.Join(root, "figures"), 0755)
	if err := os.WriteFile(filepath.Join(root, "results", "growth_bench.txt"), out, 0644); err != nil {
		log.Fatal(err)
	}

	series := parse(string(out))
	figDir := filepath.Join(root, "figures")

	lineChart(series["BenchmarkGrowthPH_Build"], "Perfect hash: Build — время vs N",
		"N, ключей", "мс/op", filepath.Join(figDir, "growth_ph_build_time.png"), true,
		func(s sample) float64 { return s.nsOp / 1e6 })

	lineChart(series["BenchmarkGrowthPH_Build"], "Perfect hash: Build — аллокации vs N",
		"N, ключей", "KiB/op", filepath.Join(figDir, "growth_ph_build_mem.png"), true,
		func(s sample) float64 { return s.bytesB / 1024 })

	lineChart(series["BenchmarkGrowthPH_Get"], "Perfect hash: Get — время vs N",
		"N, ключей в индексе", "нс/op", filepath.Join(figDir, "growth_ph_get_time.png"), true,
		func(s sample) float64 { return s.nsOp })

	lineChart(series["BenchmarkGrowthPH_Get"], "Perfect hash: Get — байты/op vs N",
		"N, ключей", "KiB/op", filepath.Join(figDir, "growth_ph_get_mem.png"), true,
		func(s sample) float64 { return s.bytesB / 1024 })

	lineChart(series["BenchmarkGrowthLSH_Find"], "LSH: FindDuplicates — время vs N",
		"N, точек", "мс/op", filepath.Join(figDir, "growth_lsh_find_time.png"), true,
		func(s sample) float64 { return s.nsOp / 1e6 })

	lineChart(series["BenchmarkGrowthLSH_Find"], "LSH: FindDuplicates — память vs N",
		"N, точек", "KiB/op", filepath.Join(figDir, "growth_lsh_find_mem.png"), true,
		func(s sample) float64 { return s.bytesB / 1024 })

	lineChart(series["BenchmarkGrowthLSH_Naive"], "Наивный полный скан — время vs N",
		"N, точек", "мс/op", filepath.Join(figDir, "growth_naive_time.png"), true,
		func(s sample) float64 { return s.nsOp / 1e6 })

	lineChart(series["BenchmarkGrowthLSH_Naive"], "Наивный полный скан — память vs N",
		"N, точек", "KiB/op", filepath.Join(figDir, "growth_naive_mem.png"), true,
		func(s sample) float64 { return s.bytesB / 1024 })

	dualTime(
		series["BenchmarkGrowthLSH_Find"],
		series["BenchmarkGrowthLSH_Naive"],
		"Поиск дублей: LSH vs наивный (логарифмические оси)",
		"N, точек", "мс/op",
		filepath.Join(figDir, "growth_dup_compare_time.png"),
	)

	multiLine(
		[]namedSeries{
			{"Set", series["BenchmarkGrowthDisk_Set"]},
			{"Get", series["BenchmarkGrowthDisk_Get"]},
		},
		"Дисковая таблица: Set и Get vs N (префилл / цикл ключей)",
		"N", "мкс/op",
		filepath.Join(figDir, "growth_disk_time.png"),
	)

	fmt.Println("PNG -> figures/growth_*.png")
}

type namedSeries struct {
	name string
	pts  []sample
}

func parse(text string) map[string][]sample {
	m := make(map[string][]sample)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		sub := lineRe.FindStringSubmatch(line)
		if sub == nil {
			continue
		}
		base := sub[1]
		n, _ := strconv.ParseFloat(sub[2], 64)
		ns, _ := strconv.ParseFloat(sub[3], 64)
		b, _ := strconv.ParseFloat(sub[4], 64)
		m[base] = append(m[base], sample{n: n, nsOp: ns, bytesB: b})
	}
	for k := range m {
		sort.Slice(m[k], func(i, j int) bool { return m[k][i].n < m[k][j].n })
	}
	return m
}

func lineChart(pts []sample, title, xlabel, ylabel, path string, logX bool, yfn func(sample) float64) {
	if len(pts) == 0 {
		return
	}
	xys := make(plotter.XYs, len(pts))
	for i, p := range pts {
		xys[i].X = p.n
		xys[i].Y = yfn(p)
	}
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xlabel
	p.Y.Label.Text = ylabel
	if logX {
		p.X.Scale = plot.LogScale{}
	}
	ln, err := plotter.NewLine(xys)
	if err != nil {
		log.Fatal(err)
	}
	ln.Color = color.RGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}
	ln.Width = vg.Points(2)
	p.Add(ln)
	if err := p.Save(10*vg.Inch, 5*vg.Inch, path); err != nil {
		log.Fatal(err)
	}
}

func dualTime(a, b []sample, title, xlabel, ylabel, path string) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xlabel
	p.Y.Label.Text = ylabel
	p.X.Scale = plot.LogScale{}
	p.Y.Scale = plot.LogScale{}

	add := func(pts []sample, col color.RGBA, name string) {
		if len(pts) == 0 {
			return
		}
		xys := make(plotter.XYs, len(pts))
		for i, s := range pts {
			xys[i].X = s.n
			xys[i].Y = s.nsOp / 1e6
		}
		ln, err := plotter.NewLine(xys)
		if err != nil {
			log.Fatal(err)
		}
		ln.Color = col
		ln.Width = vg.Points(2)
		p.Add(ln)
		p.Legend.Add(name, ln)
	}
	add(a, color.RGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, "LSH")
	add(b, color.RGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, "Naive")
	p.Legend.Top = true
	if err := p.Save(10*vg.Inch, 5*vg.Inch, path); err != nil {
		log.Fatal(err)
	}
}

func multiLine(list []namedSeries, title, xlabel, ylabel, path string) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xlabel
	p.Y.Label.Text = ylabel
	p.X.Scale = plot.LogScale{}
	p.Y.Scale = plot.LogScale{}
	cols := []color.RGBA{
		{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
		{R: 0x05, G: 0x96, B: 0x69, A: 0xff},
		{R: 0xd9, G: 0x77, B: 0x06, A: 0xff},
	}
	for i, ns := range list {
		if len(ns.pts) == 0 {
			continue
		}
		xys := make(plotter.XYs, len(ns.pts))
		for j, s := range ns.pts {
			xys[j].X = s.n
			xys[j].Y = s.nsOp / 1e3
		}
		ln, err := plotter.NewLine(xys)
		if err != nil {
			log.Fatal(err)
		}
		ln.Color = cols[i%len(cols)]
		ln.Width = vg.Points(2)
		p.Add(ln)
		p.Legend.Add(ns.name, ln)
	}
	p.Legend.Top = true
	if err := p.Save(10*vg.Inch, 5*vg.Inch, path); err != nil {
		log.Fatal(err)
	}
}
