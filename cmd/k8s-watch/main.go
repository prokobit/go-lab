package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	namespace := flag.String("namespace", "", "Namespace")
	labelSelector := flag.String("labelSelector", "", "LabelSelector")
	flag.Parse()

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

	// Set up the watch for deployments.
	dWatcher, err := clientset.AppsV1().Deployments(*namespace).Watch(ctx, metav1.ListOptions{LabelSelector: *labelSelector})
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
			case event := <-dWatcher.ResultChan():
				d, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					fmt.Println("Cannot cast event object to type appsv1.Deployment")
					continue
				}
				switch event.Type {
				case watch.Added: //, watch.Modified:
					fmt.Printf("Deployment %s added.\n", d.Name)
					image := d.Spec.Template.Spec.Containers[0].Image
					fmt.Printf("Deployment %s added with image version %s\n", d.Name, strings.Split(image, ":")[1])
				case watch.Deleted:
					fmt.Printf("Deployment %s deleted.\n", d.Name)
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
