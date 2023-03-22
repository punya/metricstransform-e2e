package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	address := flag.String("address", ":8080", "")
	metricName := flag.String("metric-name", "themetric", "")
	labelName := flag.String("label-name", "thelabel", "")
	labelValue := flag.String("label-value", "a", "")
	metricValue := flag.Float64("metric-value", 42.0, "")
	flag.Parse()

	reg := prometheus.NewRegistry()
	foo := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: *metricName,
	}, []string{*labelName})
	if err := reg.Register(foo); err != nil {
		log.Fatal(err)
	}

	c, err := foo.GetMetricWithLabelValues(*labelValue)
	if err != nil {
		log.Fatal(err)
	}
	c.Add(*metricValue)

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
	log.Fatal(http.ListenAndServe(*address, handler))
}
