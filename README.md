# ps-exporter
ps-exporter is a tool designed for Linux that collects of running processes including CPU and RAM. Formula to obtain this data is "ps aux | awk '{print $11,$3,$4}' | grep -v "0.0 0.0" | tail -n +2", and then it will convert to metrics.

## Usage
```
./ps-exporter --web.listen-address=:9123
```

## Screenshot

![Grafana](/image/grafana.png "")