go-metrics
==========

![travis build status](https://travis-ci.org/bitwalker/go-metrics.svg?branch=master)

Go port of Coda Hale's Metrics library: <https://github.com/dropwizard/metrics>.

Documentation: <http://godoc.org/github.com/bitwalker/go-metrics>.

Usage
-----

Create and update metrics:

```go
c := metrics.NewCounter()
metrics.Register("foo", c)
c.Inc(47)

g := metrics.NewGauge()
metrics.Register("bar", g)
g.Update(47)

s := metrics.NewExpDecaySample(1028, 0.015) // or metrics.NewUniformSample(1028)
h := metrics.NewHistogram(s)
metrics.Register("baz", h)
h.Update(47)

m := metrics.NewMeter()
metrics.Register("quux", m)
m.Mark(47)

t := metrics.NewTimer()
metrics.Register("bang", t)
t.Time(func() {})
t.Update(47)
```

Periodically log every metric in human-readable form to standard error:

```go
go metrics.Log(metrics.DefaultRegistry, 5 * time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
```

Periodically log every metric in slightly-more-parseable form to syslog:

```go
w, _ := syslog.Dial("unixgram", "/dev/log", syslog.LOG_INFO, "metrics")
go metrics.Syslog(metrics.DefaultRegistry, 60e9, w)
```

Maintain all metrics along with expvars at `/debug/metrics`:

This uses the same mechanism as [the official expvar](http://golang.org/pkg/expvar/)
but exposed under `/debug/metrics`, which shows a json representation of all your usual expvars
as well as all your go-metrics.


```go
import "github.com/bitwalker/go-metrics"
import "github.com/bitwalker/go-metrics/exp"

// Will use http.DefaultServeMux
exp.Exp(metrics.DefaultRegistry)

---- or use your own router

import "github.com/gorilla/mux"
import "github.com/bitwalker/go-metrics"
import "github.com/bitwalker/go-metrics/exp"

router = mux.NewRouter()
expHandler := exp.ExpHandler(metrics.DefaultRegistry)
router.HandleFunc("/debug/metrics", expHandler).Methods("GET")

```

Installation
------------

```sh
go get github.com/bitwalker/go-metrics
```
