# kube-configmap-watcher

A lightweight Go-based tool that monitors Kubernetes ConfigMap events in real-time using the client-go informer framework

## Features

- Watches ConfigMap add, update, and delete events
- Logs meaningful event info across all namespaces
- Graceful shutdown support

## Usage

```bash
go build -o kube-configmap-watcher
./kube-configmap-watcher -kubeconfig=/path/to/kubeconfig
```

> If running inside a Kubernetes cluster, the `-kubeconfig` flag is optional.
