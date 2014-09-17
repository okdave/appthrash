package memc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"appengine"
	"appengine/memcache"

	"github.com/adg/sched"
	"github.com/mjibson/appstats"
)

func init() {
	http.HandleFunc("/", handle)
	http.Handle("/mem", appstats.NewHandler(handleMem))
	http.HandleFunc("/think", handleThink)
	http.HandleFunc("/chan", handleChan)
	http.HandleFunc("/stats", handleStats)
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
	}

	sched.Check(appengine.NewContext(r))
	w.Header().Set("content-type", "text/plain; charset=utf-brad")
	w.Write([]byte(`Paths:
	/mem?count=N – fire N concurrent memcache.Get requests
		add qps=X param to add rate limit (default 20kHz)
	/think?count=N – Nk iterations of thinking
	/chan?count=N – pass a bool through chain of N channels
	/stats – output of runtime.ReadMemStats
	`))

}

func handleMem(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	sched.Check(c)

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil || count < 1 {
		http.Error(w, "Bad value for count param", http.StatusBadRequest)
		return
	}

	qps, err := strconv.Atoi(r.FormValue("qps"))
	if err != nil || qps < 1 {
		qps = 20000
	}
	throttle := time.Tick(time.Second / time.Duration(qps))

	var wg sync.WaitGroup
	wg.Add(count)
	dc := make(chan time.Duration)
	for i := 0; i < count; i++ {
		<-throttle // rate limit
		go func(i int) {
			tic := time.Now()
			_, err := memcache.Get(c, "go-william")
			dc <- time.Now().Sub(tic)
			if err != nil && err != memcache.ErrCacheMiss {
				c.Errorf("Count %d: %v", i, err)
			}
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(dc)
	}()

	alld := make(durations, 0, count)
	for d := range dc {
		alld = append(alld, d)
	}
	sort.Sort(alld)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	// Dodgy quartiles are best quartiles.
	c.Infof("count %d; quartiles [%v, %v, %v, %v, %v]", count, alld[0], alld[count/4], alld[count/2], alld[3*count/4], alld[count-1])
	fmt.Fprintf(w, "count %d; qps %d; quartiles [%v, %v, %v, %v, %v]\n", count, qps, alld[0], alld[count/4], alld[count/2], alld[3*count/4], alld[count-1])
}

func handleThink(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	sched.Check(c)

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil || count < 1 {
		http.Error(w, "Bad value for count param", http.StatusBadRequest)
		return
	}

	tic := time.Now()
	for i := 0; i < count; i++ {
		a := complex(rand.Float64(), rand.Float64())
		var z complex128
		for j := 0; j < 1000; j++ {
			z = z*z + a
		}
	}
	d := time.Now().Sub(tic)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	c.Infof("thought for %v", d)
	fmt.Fprintf(w, "thought for %v\n", d)
}

func handleChan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	sched.Check(c)

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil || count < 1 {
		http.Error(w, "Bad value for count param", http.StatusBadRequest)
		return
	}

	tic := time.Now()

	ch := make(chan bool)
	go func(chan bool) {
		ch <- true
	}(ch)
	for i := 0; i < count; i++ {
		ch1 := make(chan bool)
		go func(ch, ch1 chan bool) {
			ch1 <- <-ch
		}(ch, ch1)
		ch = ch1
	}
	<-ch
	d := time.Now().Sub(tic)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	c.Infof("channel tunnel took %v", d)
	fmt.Fprintf(w, "channel tunnel took %v\n", d)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	sched.Check(c)

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	b, err := json.MarshalIndent(stats, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Write(b)
}

type durations []time.Duration

func (d durations) Len() int           { return len(d) }
func (d durations) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d durations) Less(i, j int) bool { return d[i] < d[j] }
