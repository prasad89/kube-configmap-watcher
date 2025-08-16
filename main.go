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
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	configMapInformer cache.SharedIndexInformer
	podInformer       cache.SharedIndexInformer
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

	// Get informers
	configMapInformer = informerFactory.Core().V1().ConfigMaps().Informer()
	podInformer = informerFactory.Core().V1().Pods().Informer()

	// Add indexer on Pods to get configMap ref
	err = podInformer.AddIndexers(cache.Indexers{
		"configMapRef": func(obj any) ([]string, error) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return nil, nil
			}

			var keys []string

			ns := pod.Namespace

			// Volume ConfigMap refs
			for _, vol := range pod.Spec.Volumes {
				if vol.ConfigMap != nil {
					keys = append(keys, ns+"/"+vol.ConfigMap.Name)
				}
			}

			// EnvFrom ConfigMap refs
			for _, envFrom := range pod.Spec.Containers {
				for _, source := range envFrom.EnvFrom {
					if source.ConfigMapRef != nil {
						keys = append(keys, ns+"/"+source.ConfigMapRef.Name)
					}
				}
			}

			// Env ConfigMap refs
			for _, c := range pod.Spec.Containers {
				for _, e := range c.Env {
					if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
						keys = append(keys, ns+"/"+e.ValueFrom.ConfigMapKeyRef.Name)
					}
				}
			}

			return keys, nil
		},
	})
	if err != nil {
		log.Fatalf("Error adding pod indexer: %v", err)
	}

	// Register event handlers
	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onConfigMapAdd,
		UpdateFunc: onConfigMapUpdate,
		DeleteFunc: onConfigMapDelete,
	})

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onPodAdd,
		UpdateFunc: onPodUpdate,
		DeleteFunc: onPodDelete,
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
	log.Println("Starting informers...")
	informerFactory.Start(stopCh)

	// Wait for all caches to sync
	if ok := cache.WaitForCacheSync(stopCh, configMapInformer.HasSynced, podInformer.HasSynced); !ok {
		runtime.HandleError(err)
		log.Fatal("Failed to sync caches")
	}

	log.Println("Informers running")
	<-ctx.Done()
	log.Println("Controller stopped")
}

func onConfigMapAdd(obj any) {
	if cm, ok := obj.(*v1.ConfigMap); ok {
		log.Printf("[ADD] ConfigMap: %s/%s", cm.Namespace, cm.Name)
	}
}

func onConfigMapUpdate(oldObj, newObj any) {
	cm, ok := newObj.(*v1.ConfigMap)
	if !ok {
		return
	}
	log.Printf("[UPDATE] ConfigMap: %s/%s", cm.Namespace, cm.Name)

	key := cm.Namespace + "/" + cm.Name
	pods, err := podInformer.GetIndexer().ByIndex("configMapRef", key)
	if err != nil {
		log.Printf("Error fetching pods from index: %v", err)
		return
	}

	log.Printf("Found %d Pods using this ConfigMap:", len(pods))
	for _, obj := range pods {
		if pod, ok := obj.(*v1.Pod); ok {
			log.Printf(" - %s/%s", pod.Namespace, pod.Name)
		}
	}
}

func onConfigMapDelete(obj any) {
	var cm *v1.ConfigMap
	switch obj := obj.(type) {
	case *v1.ConfigMap:
		cm = obj
	case cache.DeletedFinalStateUnknown:
		cm, _ = obj.Obj.(*v1.ConfigMap)
	}
	if cm != nil {
		log.Printf("[DELETE] ConfigMap: %s/%s", cm.Namespace, cm.Name)
	}
}

func onPodAdd(obj any) {
	if pod, ok := obj.(*v1.Pod); ok {
		log.Printf("[ADD] Pod: %s/%s", pod.Namespace, pod.Name)
	}
}

func onPodUpdate(oldObj, newObj any) {
	if pod, ok := newObj.(*v1.Pod); ok {
		log.Printf("[UPDATE] Pod: %s/%s", pod.Namespace, pod.Name)
	}
}

func onPodDelete(obj any) {
	var pod *v1.Pod
	switch obj := obj.(type) {
	case *v1.Pod:
		pod = obj
	case cache.DeletedFinalStateUnknown:
		pod, _ = obj.Obj.(*v1.Pod)
	}
	if pod != nil {
		log.Printf("[DELETE] Pod: %s/%s", pod.Namespace, pod.Name)
	}
}
