package metrics

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	// statsdMaxLen is the maximum size of a packet to send to statsd
	statsdMaxLen = 1400
	// We force flush the statsd metrics after this period
	statsdFlushInterval = 100 * time.Millisecond
)

type statsdSink struct {
	metricQueue chan string
	addr        string
	duration    time.Duration
}

func Statsd(r Registry, d time.Duration, addr string) {
	sink := &statsdSink{
		metricQueue: make(chan string, 4096),
		addr:        addr,
		duration:    d,
	}
	go sink.flushMetrics()

	for {
		r.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case Counter:
				sink.pushMetric(fmt.Sprintf("%s:%d|kv\n", name, int(metric.Count())))
			case Gauge:
				sink.pushMetric(fmt.Sprintf("%s:%f|kv\n", name, float64(metric.Value())))
			case GaugeFloat64:
				sink.pushMetric(fmt.Sprintf("%s:%f|kv\n", name, float64(metric.Value())))
			case Histogram:
				ss := metric.Snapshot()
				ps := ss.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				sink.pushMetric(fmt.Sprintf("%s.count:%d|kv\n", name, int(ss.Count())))
				sink.pushMetric(fmt.Sprintf("%s.min:%f|kv\n", name, float64(ss.Min())))
				sink.pushMetric(fmt.Sprintf("%s.max:%f|kv\n", name, float64(ss.Max())))
				sink.pushMetric(fmt.Sprintf("%s.mean:%f|kv\n", name, float64(ss.Mean())))
				sink.pushMetric(fmt.Sprintf("%s.std-dev:%f|kv\n", name, float64(ss.StdDev())))
				sink.pushMetric(fmt.Sprintf("%s.50-percentile:%f|kv\n", name, float64(ps[0])))
				sink.pushMetric(fmt.Sprintf("%s.75-percentile:%f|kv\n", name, float64(ps[1])))
				sink.pushMetric(fmt.Sprintf("%s.95-percentile:%f|kv\n", name, float64(ps[2])))
				sink.pushMetric(fmt.Sprintf("%s.99-percentile:%f|kv\n", name, float64(ps[3])))
				sink.pushMetric(fmt.Sprintf("%s.999-percentile:%f|kv\n", name, float64(ps[4])))
			case Meter:
				ss := metric.Snapshot()
				sink.pushMetric(fmt.Sprintf("%s.count:%d|kv\n", name, int(ss.Count())))
				sink.pushMetric(fmt.Sprintf("%s.one-minute:%d|kv\n", name, float64(ss.Rate1())))
				sink.pushMetric(fmt.Sprintf("%s.five-minute:%d|kv\n", name, float64(ss.Rate5())))
				sink.pushMetric(fmt.Sprintf("%s.fifteen-minute:%d|kv\n", name, float64(ss.Rate15())))
				sink.pushMetric(fmt.Sprintf("%s.mean:%d|kv\n", name, float64(ss.RateMean())))
			case Timer:
				ss := metric.Snapshot()
				ps := ss.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				sink.pushMetric(fmt.Sprintf("%s.count:%d|kv\n", name, int(ss.Count())))
				sink.pushMetric(fmt.Sprintf("%s.min:%f|kv\n", name, float64(ss.Min())))
				sink.pushMetric(fmt.Sprintf("%s.max:%f|kv\n", name, float64(ss.Max())))
				sink.pushMetric(fmt.Sprintf("%s.mean:%f|kv\n", name, float64(ss.Mean())))
				sink.pushMetric(fmt.Sprintf("%s.std-dev:%f|kv\n", name, float64(ss.StdDev())))
				sink.pushMetric(fmt.Sprintf("%s.50-percentile:%f|kv\n", name, float64(ps[0])))
				sink.pushMetric(fmt.Sprintf("%s.75-percentile:%f|kv\n", name, float64(ps[1])))
				sink.pushMetric(fmt.Sprintf("%s.95-percentile:%f|kv\n", name, float64(ps[2])))
				sink.pushMetric(fmt.Sprintf("%s.99-percentile:%f|kv\n", name, float64(ps[3])))
				sink.pushMetric(fmt.Sprintf("%s.999-percentile:%f|kv\n", name, float64(ps[4])))
				sink.pushMetric(fmt.Sprintf("%s.one-minute:%d|kv\n", name, float64(ss.Rate1())))
				sink.pushMetric(fmt.Sprintf("%s.five-minute:%d|kv\n", name, float64(ss.Rate5())))
				sink.pushMetric(fmt.Sprintf("%s.fifteen-minute:%d|kv\n", name, float64(ss.Rate15())))
				sink.pushMetric(fmt.Sprintf("%s.mean:%d|kv\n", name, float64(ss.RateMean())))
			}
		})
		time.Sleep(d)
	}
}

// Performs non-blocking push to the metrics queue
func (s *statsdSink) pushMetric(m string) {
	select {
	case s.metricQueue <- m:
	default:
	}
}

func (s *statsdSink) flushMetrics() {
	var sock net.Conn
	var err error
	var wait <-chan time.Time
	ticker := time.NewTicker(statsdFlushInterval)
	defer ticker.Stop()

CONNECT:
	// Create a buffer
	buf := bytes.NewBuffer(nil)

	// Attempt to connect
	sock, err = net.Dial("udp", s.addr)
	if err != nil {
		log.Printf("[ERR] Error connecting to statsd! Err: %s", err)
		goto WAIT
	}

	for {
		select {
		case metric, ok := <-s.metricQueue:
			// Get a metric from the queue
			if !ok {
				goto QUIT
			}

			// Check if this would overflow the packet size
			if len(metric)+buf.Len() > statsdMaxLen {
				_, err := sock.Write(buf.Bytes())
				buf.Reset()
				if err != nil {
					log.Printf("[ERR] Error writing to statsd! Err: %s", err)
					goto WAIT
				}
			}

			// Append to the buffer
			buf.WriteString(metric)

		case <-ticker.C:
			if buf.Len() == 0 {
				continue
			}

			_, err := sock.Write(buf.Bytes())
			buf.Reset()
			if err != nil {
				log.Printf("[ERR] Error flushing to statsd! Err: %s", err)
				goto WAIT
			}
		}
	}

WAIT:
	// Wait for a while
	wait = time.After(s.duration)
	for {
		select {
		// Dequeue the messages to avoid backlog
		case _, ok := <-s.metricQueue:
			if !ok {
				goto QUIT
			}
		case <-wait:
			goto CONNECT
		}
	}
QUIT:
	s.metricQueue = nil
}
