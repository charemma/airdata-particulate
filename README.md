# airdata-particulate

Prometheus exporter for the **Nova PM SDS011** particulate matter sensor. Runs on a Raspberry Pi 5 (aiagent), measures PM2.5 and PM10 concentrations, and exposes them at `/metrics` for Prometheus scraping.

**Live data:** [monitoring.charemma.de/d/airdata-particulate](https://monitoring.charemma.de/d/airdata-particulate) (public read-only dashboard)

## How the sensor works

The SDS011 uses **laser scattering** to count and size particles in a flowing air stream. A small fan pulls ambient air past a 650nm laser diode; particles crossing the beam scatter light onto a photodetector. The pulse pattern is analysed to derive concentrations in two size classes:

- **PM2.5** -- particles up to 2.5 µm (fine particulate, penetrates deep into lungs and bloodstream). Sources: combustion, cooking, wood/tobacco smoke, wildfires.
- **PM10** -- particles up to 10 µm (coarse particulate). Sources: pollen, road dust, mechanical wear, agriculture.

Values are reported in **µg/m³** (micrograms per cubic metre). EU air quality thresholds (annual mean):

| Metric | WHO 2021 guideline | EU limit |
|--------|--------------------|----------|
| PM2.5 | 5 µg/m³ | 25 µg/m³ (soon 10) |
| PM10 | 15 µg/m³ | 40 µg/m³ (soon 20) |

## Measurement cycle

To extend sensor lifetime (rated ~8000h continuous, longer duty-cycled) the exporter wakes the sensor, takes one measurement, then sleeps it again:

```
loop:
  sensor.Awake()      # spin up fan and laser
  sleep 30s            # warm-up: air flow stabilises
  point = sensor.Query()   # returns pm25, pm10
  pm25Gauge.Set(point.PM25)
  pm10Gauge.Set(point.PM10)
  sensor.Sleep()       # fan + laser off
  sleep 300s           # 5 minute idle
```

Effective duty cycle: ~30s active / 5:00 min idle -> sensor life multiplies by ~10x compared with continuous operation.

## Metrics

Both are gauges, unit µg/m³:

```
# HELP pm25 PM2.5 concentration in µg/m³
# TYPE pm25 gauge
pm25 12.3
# HELP pm10 PM10 concentration in µg/m³
# TYPE pm10 gauge
pm10 18.7
```

## Build and run

```bash
go build -o particulate .
./particulate                # needs SDS011 on /dev/ttyUSB0, prints readings to stdout
curl localhost:8000/metrics
```

Or via dagger:

```bash
dagger call lint
dagger call build            # multi-arch (linux/arm64) container image
```

CI publishes to `ghcr.io/charemma/airdata-particulate:latest` on every push to main.

## Deployment

Kubernetes manifests in `k8s/` are consumed by argocd via kustomize:

- `deployment.yml` -- runs on any node labelled `sensor-type=particulate` (aiagent), privileged for USB device access, mounts `/dev/ttyUSB0` from host
- `service.yml` + `servicemonitor.yml` -- prometheus-operator scrapes the exporter every 60s
- `dashboard/particulate.json` -- Grafana dashboard, generated into a ConfigMap by kustomize and picked up by the grafana sidecar (label `grafana_dashboard=1`) so it appears in the "Air Quality" folder without manual import

The argocd Application lives in [charemma/platform](https://github.com/charemma/platform) at `gitops/apps/particulate.yaml`.

## Hardware

- **Sensor:** Nova PM Sensor SDS011 (7-pin UART, USB adapter)
- **Host:** Raspberry Pi 5 (aiagent), running NixOS
- **Placement:** indoor, ground floor
