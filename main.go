package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ryszard/sds011/go/sds011"
)

var (
	pm25Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pm25",
		Help: "PM2.5 concentration in µg/m³",
	})
	pm10Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pm10",
		Help: "PM10 concentration in µg/m³",
	})
)

func main() {
	device := flag.String("device", "/dev/ttyUSB0", "Serial device the SDS011 is attached to")
	addr := flag.String("addr", ":8000", "HTTP listen address for the metrics endpoint")
	warmup := flag.Duration("warmup", 30*time.Second, "Time to wait after wake before querying the sensor")
	interval := flag.Duration("interval", 5*time.Minute, "Idle time between measurement cycles")
	flag.Parse()

	prometheus.MustRegister(pm25Gauge, pm10Gauge)

	sensor, err := sds011.New(*device)
	if err != nil {
		log.Fatalf("failed to open %s: %v", *device, err)
	}
	defer sensor.Close()

	go func() {
		for {
			if err := sensor.Awake(); err != nil {
				log.Printf("failed to wake up sensor: %v", err)
				continue
			}

			time.Sleep(*warmup)

			point, err := sensor.Query()
			if err != nil {
				log.Printf("failed to query sensor: %v", err)
			} else {
				pm25Gauge.Set(point.PM25)
				pm10Gauge.Set(point.PM10)
				log.Printf("PM2.5: %.1f, PM10: %.1f", point.PM25, point.PM10)
			}

			if err := sensor.Sleep(); err != nil {
				log.Printf("failed to put sensor to sleep: %v", err)
			}

			time.Sleep(*interval)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("[particulate-exporter] device=%s listen=%s", *device, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
