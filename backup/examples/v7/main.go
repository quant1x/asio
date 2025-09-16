package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	url         string
	concurrency int
	total       int
	timeout     time.Duration
)

func init() {
	flag.StringVar(&url, "url", "http://localhost:8080", "Target URL")
	flag.IntVar(&concurrency, "c", 100, "Concurrency level")
	flag.IntVar(&total, "n", 10000, "Total number of requests")
	flag.DurationVar(&timeout, "t", 10*time.Second, "Request timeout")
}

type result struct {
	statusCode int
	duration   time.Duration
	err        error
}

func main() {
	flag.Parse()

	start := time.Now()
	results := make(chan result, total)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < total/concurrency; j++ {
				reqStart := time.Now()
				statusCode, err := sendRequest()
				results <- result{
					statusCode: statusCode,
					duration:   time.Since(reqStart),
					err:        err,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var success, failure int
	var totalDuration time.Duration
	statusCodes := make(map[int]int)
	for r := range results {
		if r.err != nil {
			failure++
		} else {
			success++
			statusCodes[r.statusCode]++
		}
		totalDuration += r.duration
	}

	elapsed := time.Since(start)
	showResults(elapsed, success, failure, statusCodes, totalDuration)
}

func sendRequest() (int, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	//start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 读取响应体以确保连接可复用
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

func showResults(elapsed time.Duration, success, failure int, statusCodes map[int]int, totalDuration time.Duration) {
	fmt.Printf("\n测试结果:\n")
	fmt.Printf("总请求数: %d\n", success+failure)
	fmt.Printf("成功: %d\n", success)
	fmt.Printf("失败: %d\n", failure)
	fmt.Printf("总耗时: %s\n", elapsed)
	fmt.Printf("请求吞吐量: %.2f req/s\n", float64(success+failure)/elapsed.Seconds())
	fmt.Printf("平均响应时间: %s\n", time.Duration(int64(totalDuration)/int64(success+failure)))

	fmt.Println("\n状态码分布:")
	for code, count := range statusCodes {
		fmt.Printf("  %d: %d\n", code, count)
	}
}
