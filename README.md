# kube-configmap-watcher

A lightweight Go-based tool that monitors Kubernetes ConfigMap events in real-time using the client-go informer framework.

## Features

- 🔍 Watches ConfigMap add, update, and delete events
- 🌐 Monitors events across all namespaces  
- 📝 Clean, structured event logging
- 🛑 Graceful shutdown with signal handling
- ⚡ Built with Kubernetes informer framework for efficiency

## Prerequisites

- Go 1.24.5+ (for local development)
- Access to a Kubernetes cluster
- kubectl configured with appropriate cluster access

## Usage

### Local Development

```bash
go mod tidy
go build -o configmap-watcher
./configmap-watcher -kubeconfig=/path/to/kubeconfig
```

### Deploy to Kubernetes

The included manifest creates all necessary RBAC resources and deploys the watcher:

```bash
kubectl apply -f configmap-watcher.yaml
```

To view logs:

```bash
kubectl logs -f deployment/configmap-watcher -n configmap-watcher
```

> **Note:** When running inside a Kubernetes cluster, the `-kubeconfig` flag is optional as it uses in-cluster configuration.
