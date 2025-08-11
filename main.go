package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Parse kubeconfig flag
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (optional if running in cluster)")
	flag.Parse()

	// Build config from flags
	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error creating Kubernetes clientset: %v", err)
	}

	// Create shared informer factory with resync period
	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	// Get ConfigMap informer
	configMapInformer := informerFactory.Core().V1().ConfigMaps()

	// Register event handlers
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	// Set up signal handling and context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
		close(stopCh)
	}()

	// Start informers
	informerFactory.Start(stopCh)

	// Wait for all caches to sync
	if ok := cache.WaitForCacheSync(stopCh, configMapInformer.Informer().HasSynced); !ok {
		log.Fatalf("Failed to sync caches")
	}

	log.Println("Starting ConfigMap informer")
	<-ctx.Done()
	log.Println("Informer stopped")
}

func onAdd(obj interface{}) {
	if cm, ok := obj.(*v1.ConfigMap); ok {
		log.Printf("[ADD] ConfigMap: %s/%s", cm.Namespace, cm.Name)
	}
}

func onUpdate(oldObj, newObj interface{}) {
	if cm, ok := newObj.(*v1.ConfigMap); ok {
		log.Printf("[UPDATE] ConfigMap: %s/%s", cm.Namespace, cm.Name)
	}
}

func onDelete(obj interface{}) {
	var cm *v1.ConfigMap

	switch obj := obj.(type) {
	case *v1.ConfigMap:
		cm = obj
	case cache.DeletedFinalStateUnknown:
		var ok bool
		cm, ok = obj.Obj.(*v1.ConfigMap)
		if !ok {
			log.Printf("[DELETE] Unknown object type: %+v", obj.Obj)
			return
		}
	default:
		log.Printf("[DELETE] Unknown object type: %+v", obj)
		return
	}

	log.Printf("[DELETE] ConfigMap: %s/%s", cm.Namespace, cm.Name)
}
