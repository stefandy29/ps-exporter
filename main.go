package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Ps() string {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	var stdout bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", `ps aux | awk '{print $11,$3,$4}' | grep -v "0.0 0.0" | tail -n +2`)
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		log.Println(err)
		return ""
	}
	output := stdout.Bytes()
	return string(output)
}

type usageCollector struct {
	cpu_usage *prometheus.Desc
	ram_usage *prometheus.Desc
}

func newUsageCollector() *usageCollector {
	return &usageCollector{
		cpu_usage: prometheus.NewDesc("ps_cpu_process",
			"cpu",
			[]string{"process_name"}, nil,
		),
		ram_usage: prometheus.NewDesc("ps_ram_process",
			"ram",
			[]string{"process_name"}, nil,
		),
	}
}

func (collector *usageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.cpu_usage
	ch <- collector.ram_usage
}

func (collector *usageCollector) Collect(ch chan<- prometheus.Metric) {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	var wg sync.WaitGroup
	cpu_data := make(map[string]float64)
	ram_data := make(map[string]float64)
	all_data := Ps()
	if len(all_data) == 0 {
		log.Println("data not found")
		return
	}
	if !regexp.MustCompile(`\n`).MatchString(all_data) {
		return
	}
	value_all := regexp.MustCompile(`\n`).Split(all_data, -1)
	for _, value := range value_all {
		value_row := regexp.MustCompile(" +").Split(value, -1)
		if len(value_row) == 3 {
			cpu, err := strconv.ParseFloat(value_row[1], 64)
			if err != nil {
				cpu = 0
			}
			ram, err := strconv.ParseFloat(value_row[2], 64)
			if err != nil {
				ram = 0
			}
			if val, err := cpu_data[string(value_row[0])]; err {
				cpu_data[string(value_row[0])] = val + cpu
			} else {
				cpu_data[string(value_row[0])] = cpu
			}

			if val, err := ram_data[string(value_row[0])]; err {
				ram_data[string(value_row[0])] = val + ram
			} else {
				ram_data[string(value_row[0])] = ram
			}
		}
	}
	for key, value := range cpu_data {
		wg.Add(1)
		go func(key string, value float64) {
			defer wg.Done()
			query := prometheus.MustNewConstMetric(
				collector.cpu_usage,
				prometheus.GaugeValue,
				math.Round(value*100)/100,
				key)
			query = prometheus.NewMetricWithTimestamp(time.Now(), query)
			ch <- query
		}(key, value)
	}
	wg.Wait()

	for key, value := range ram_data {
		wg.Add(1)
		go func(key string, value float64) {
			defer wg.Done()
			query := prometheus.MustNewConstMetric(
				collector.ram_usage,
				prometheus.GaugeValue,
				math.Round(value*100)/100,
				key)
			query = prometheus.NewMetricWithTimestamp(time.Now(), query)
			ch <- query
		}(key, value)
	}
	wg.Wait()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var addr string
	flag.StringVar(&addr, "web.listen-address", "", "The address to listen on for HTTP requests.")
	flag.Parse()
	if len(addr) == 0 {
		log.Println("Usage: \n./ps-exporter --web.listen-address=:8080")
		flag.PrintDefaults()
		os.Exit(1)
	}
	prometheus.MustRegister(newUsageCollector())
	prometheus.Unregister(collectors.NewGoCollector())
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting ps exporter at port %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
