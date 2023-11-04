package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

	// access kubernetes config
	var home = homedir.HomeDir()
	var kubeconfig = filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	// ensure kubernetes is configured
	if err != nil {
		panic(err.Error())
	}

	// Configure dynamic kubeclient
	dyclient, err := dynamic.NewForConfig(config)

	// ensure no error in obtaining the kubernetes client
	if err != nil {
		panic(err.Error())
	}

	// configure options
	b := true
	opts := metav1.ListOptions{
		Watch:                true,
		SendInitialEvents:    &b,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}

	// attempts to begin watching the namespaces
	// returns a `watch.Interface`, or an error

	// create the group version response
	gvr, _ := schema.ParseResourceArg("pods.v1.")

	// check if WatchList feature is enable
	watchList := checkWatchListFeatureBruteForce(dyclient)

	if !watchList {
		if err != nil {
			panic(errors.New("WatchList feature not enabled"))
		}
	}

	// create the watch command
	watcher, err := dyclient.Resource(*gvr).Watch(context.TODO(), opts)

	// Create a stop channel to signal the goroutine to stop
	stopChannel := make(chan struct{})

	// create watch group for synchronizing go routines
	var wg sync.WaitGroup

	// Launch the goroutine and pass the channel as an argument
	wg.Add(1)
	go backgroundProcessor(watcher.ResultChan(), stopChannel, &wg)

	wg.Wait()
}
func backgroundProcessor(result <-chan watch.Event, stopCh chan struct{}, wg *sync.WaitGroup) {

	for {

		for event := range result {
			// retrieve the Namespace
			// item := event.Object.(*unstructured.Unstructured)
			obj := event.Object.(*unstructured.Unstructured)

			switch event.Type {

			// when an event is deleted...
			case watch.Deleted:

				fmt.Println("Received DELETE event for: ", obj.GetName(), "/", obj.GetNamespace(), " of kind: ", obj.GroupVersionKind().Kind)

			// when an event is added...
			case watch.Added:
				fmt.Println("Received ADD event for: ", obj.GetName(), "/", obj.GetNamespace(), " of kind: ", obj.GroupVersionKind().Kind)

				// when an event is added...
			case watch.Modified:
				fmt.Println("Received UPDATE event for: ", obj.GetName(), "/", obj.GetNamespace(), " of kind: ", obj.GroupVersionKind().Kind)

			}

			// check if channel is closed
			if len(stopCh) > 0 {
				println("Goroutines killed!")
				wg.Done()
				return
			}
		}
	}
}

// checkWatchListFeatureOs checks whether the WatchList feature gate is enabled
// by doing a ps aux command and matching the output with 'WatchList=true' string that would signify
// the feature being set
func checkWatchListFeatureOs() bool {
	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		print(err.Error())
	}
	r, _ := regexp.Compile("WatchList=true")

	if r.MatchString(out.String()) {
		return true
	}

	return false
}

// checkWatchListFeatureBruteForce checks if the WatchList feature is present by doing a test
// streaming list watch command on a simple pod and watching the result, a positive result
// means the feature is enabled
func checkWatchListFeatureBruteForce(client dynamic.Interface) bool {
	b := true
	opts := metav1.ListOptions{
		Watch:                true,
		SendInitialEvents:    &b,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}
	gvr, _ := schema.ParseResourceArg("pods.v1.")
	_, err := client.Resource(*gvr).Watch(context.TODO(), opts)

	return err == nil
}
