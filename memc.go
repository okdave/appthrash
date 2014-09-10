package memc

import (
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"appengine"
	"appengine/memcache"

	"github.com/adg/sched"
)

func init() {
	http.HandleFunc("/", handle)
	http.HandleFunc("/mem", handleMem)
	http.HandleFunc("/think", handleThink)
	http.HandleFunc("/chan", handleChan)
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
	}

	sched.Check(appengine.NewContext(r))
	w.Header().Set("content-type", "text/plain; charset=utf-brad")
	w.Write([]byte(`Paths:
	/mem?count=N – fire N concurrent memcache.Get requests
	/think?count=N – Nk iterations of thinking
	/chan?count=N – pass a bool through chain of N channels
	`))

}

func handleMem(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	sched.Check(c)

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil || count < 1 {
		http.Error(w, "Bad value for count param", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	wg.Add(count)
	dc := make(chan time.Duration)
	for i := 0; i < count; i++ {
		go func(i int) {
			tic := time.Now()
			_, err := memcache.Get(c, "roger")
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
	fmt.Fprintf(w, "count %d; quartiles [%v, %v, %v, %v, %v]", count, alld[0], alld[count/4], alld[count/2], alld[3*count/4], alld[count-1])
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
	fmt.Fprintf(w, "thought for %v", d)
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
	fmt.Fprintf(w, "channel tunnel took %v", d)
}

type durations []time.Duration

func (d durations) Len() int           { return len(d) }
func (d durations) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d durations) Less(i, j int) bool { return d[i] < d[j] }
