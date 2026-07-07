# airdata-particulate

Prometheus exporter for the **Nova PM SDS011** particulate matter sensor. Reads over a UART/USB serial link, exposes PM2.5 and PM10 concentrations at `/metrics`.

**Live data:** [public grafana dashboard](https://monitoring.charemma.de/public-dashboards/3f3e0fdc8d99483e907690354824eeb6)

## How the sensor works

The SDS011 uses **laser scattering** to count and size particles in a flowing air stream. A small fan pulls ambient air past a 650 nm laser diode; particles crossing the beam scatter light onto a photodetector. Pulse patterns are analysed to derive concentrations in two size classes:

- **PM2.5** -- particles up to 2.5 µm. Fine particulate; penetrates deep into lungs and bloodstream. Sources: combustion, cooking, wood/tobacco smoke, wildfires.
- **PM10** -- particles up to 10 µm. Sources: pollen, road dust, mechanical wear, agriculture.

Concentrations are reported in **µg/m³**. Reference thresholds (annual mean):

| | WHO 2021 guideline | EU limit |
|--|--|--|
| PM2.5 | 5 µg/m³ | 25 µg/m³ |
| PM10 | 15 µg/m³ | 40 µg/m³ |

## Measurement cycle

The SDS011 has a rated life of ~8000h continuous. To extend that, the exporter wakes the sensor, takes one measurement, then sleeps it:

```
loop:
  sensor.Awake()
  sleep 30s            # warm-up: air flow stabilises
  point = sensor.Query()
  pm25Gauge.Set(point.PM25)
  pm10Gauge.Set(point.PM10)
  sensor.Sleep()
  sleep 300s           # idle
```

Effective duty cycle: ~30s active / 5:00 min idle -- extends sensor life ~10x.

## Metrics

Both gauges, unit µg/m³:

```
# HELP pm25 PM2.5 concentration in µg/m³
# TYPE pm25 gauge
pm25 12.3
# HELP pm10 PM10 concentration in µg/m³
# TYPE pm10 gauge
pm10 18.7
```

## Build

Everything is driven by the flake. Enter a shell:

```bash
nix develop
go build -o particulate .   # native binary, quick edits
```

Or build the release binary directly:

```bash
nix build .#default
./result/bin/particulate    # exposes :8000/metrics; expects sensor on /dev/ttyUSB0
```

## Runtime flags

Nothing is hardcoded. Pass whatever fits the host:

| flag | default | notes |
|--|--|--|
| `-device` | `/dev/ttyUSB0` | serial port the SDS011 hangs on |
| `-addr` | `:8000` | HTTP listen address |
| `-warmup` | `30s` | time after wake before first query (Go duration syntax) |
| `-interval` | `5m` | idle time between measurement cycles |

Example:

```bash
particulate -device /dev/ttyUSB1 -addr :9100 -interval 2m
```

The container image inherits these defaults; override via `args:` in the Deployment.

Container image (arm64):

```bash
nix build .#docker
# result -> tarball; load into podman/docker or push with skopeo
skopeo copy docker-archive:./result docker://ghcr.io/charemma/airdata-particulate:latest
```

## Deployment

Kubernetes manifests in `k8s/` are consumed by argocd via kustomize:

- `deployment.yml` -- privileged pod, mounts `/dev/ttyUSB0` from host, pinned via `sensor-type=particulate` and `kubernetes.io/arch=arm64`
- `service.yml` + `servicemonitor.yml` -- prometheus-operator scrapes the exporter every 60s
- `dashboard/particulate.json` -- Grafana dashboard, generated into a ConfigMap by kustomize and picked up by the grafana sidecar (label `grafana_dashboard=1`) so it appears in the "Air Quality" folder

The argocd Application lives in [charemma/platform](https://github.com/charemma/platform) at `gitops/apps/particulate.yaml`.
