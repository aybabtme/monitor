package main

import (
	"flag"
	"fmt"
	"github.com/aybabtme/uniplot/barchart"
	"github.com/aybabtme/uniplot/spark"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

func main() {
	var (
		duration  = flag.Duration("dur", 10*time.Second, "duration of the test")
		maxRPS    = flag.Float64("rps", 10.0, "max requests per seconds")
		conc      = flag.Int("conc", 1, "concurrent requests")
		tgt       = flag.String("tgt", "", "target to test, must be a full http://link.com/path")
		fetchBody = flag.Bool("fetch-body", false, "fetch the body of the response")
	)
	flag.Parse()

	log.SetPrefix("monitor: ")
	log.SetFlags(0)

	if *tgt == "" {
		log.Print("need a target")
		flag.PrintDefaults()
		log.Fatal("invalid usage")
	}

	var points [][2]int

	jobRate := time.Duration(float64(time.Second) / *maxRPS)

	timeout := time.NewTimer(*duration)
	jobTicker := time.NewTicker(jobRate)

	jobc := make(chan struct{}) // no buffering == no burst when pilling up
	outc := make(chan time.Duration, *conc*2)
	errc := make(chan error, *conc*2)

	http.DefaultClient.Timeout = time.Second * 2

	var wg sync.WaitGroup
	for i := 0; i < *conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doHttpRequests(*tgt, *fetchBody, jobc, outc, errc)
		}()
	}

	point := [2]int{
		int(time.Now().Unix()),
		0,
	}

	sprk := spark.Spark(between(30*time.Millisecond, time.Second, jobRate*2))
	sprk.Units = "rps"
	sprk.Start()
loop:
	for {
		select {
		case res := <-outc:
			sprk.Add(1)
			t := time.Now()
			if int(t.Unix()) != point[0] {
				points = append(points, point)
				point = [2]int{
					int(t.Unix()),
					1,
				}
				continue
			}
			_ = res
			point[1]++
		case <-jobTicker.C:
			select {
			case jobc <- struct{}{}:
			default:
			}
		case err := <-errc:
			log.Printf("error: %v", err)
		case <-timeout.C:
			close(jobc)
			sprk.Stop()
			break loop
		}
	}
	wg.Wait()
	close(errc)
	close(outc)

	barchart.Fprintf(
		os.Stdout,
		barchart.BarChartXYs(points),
		len(points),
		barchart.Linear(40),
		func(x float64) string {
			return time.Unix(int64(x), 0).String()
		},
		func(y float64) string {
			return fmt.Sprintf("%d req/s", int(y))
		},
	)
}

func doHttpRequests(tgt string, fetchBody bool, jobc <-chan struct{}, outc chan<- time.Duration, errc chan<- error) {
	for _ = range jobc {
		start := time.Now()
		resp, err := http.Get(tgt)
		if err != nil {
			errc <- err
			continue
		}
		if fetchBody {
			_, err := io.Copy(ioutil.Discard, resp.Body)
			if err != nil {
				_ = resp.Body.Close()
				errc <- err
				continue
			}
		}
		err = resp.Body.Close()
		if err != nil {
			errc <- err
			continue
		}
		outc <- time.Since(start)
	}
}

func between(min, max, val time.Duration) time.Duration {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
