package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// Find the kubeconfig file.
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	// Load kubeconfig file.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create a Kubernetes client.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Create a context for the watch.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the watch for pods.
	podWatcher, err := clientset.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Set up a channel to receive interrupts.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle watch events.
	go func() {
		for {
			select {
			case event := <-podWatcher.ResultChan():
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					fmt.Println("Cannot cast event object to type corev1.Pod")
					continue
				}
				switch event.Type {
				case watch.Added: //, watch.Modified:
					image := pod.Spec.Containers[0].Image
					fmt.Printf("Pod %s added with image version %s\n", pod.Name, strings.Split(image, ":")[1])
				case watch.Deleted:
					fmt.Printf("Pod %s deleted.\n", pod.Name)
				}
			case <-interrupt:
				// Stop the watch on interrupt.
				cancel()
				return
			}
		}
	}()

	// Block until interrupted.
	<-interrupt
	fmt.Println("Watch stopped.")
}
