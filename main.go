package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/myntra/pipeline"
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

	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		print(err.Error())
	}
	r, _ := regexp.Compile("WatchList=true")

	if r.MatchString(out.String()) {
		print("Matches")
	}
	print(out.String())
	// AUTHENTICATE
	var home = homedir.HomeDir()
	var kubeconfig = filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {

		panic(err.Error())
	}

	// Configure dynamic kubeclient
	dyclient, err := dynamic.NewForConfig(config)
	// clusterPipeline := pipeline.New("pods", 1000)
	// podStage := &pipeline.Stage{
	// 	Name:       "process pods",
	// 	Concurrent: false,
	// 	Steps:      []pipeline.Step{},
	// }
	// // Create a stop channel to signal the goroutine to stop
	// stopChan := make(chan struct{})
	// podStage.AddStep(newStartWatcherStage(*dyclient, "pods.v1.", stopChan))
	// clusterPipeline.AddStage(podStage)
	// _ = clusterPipeline.Run()
	// if err != nil {
	// }

	// we want to use the core API (namespaces lives here)
	// config.APIPath = "/api"
	// config.GroupVersion = &coreV1.SchemeGroupVersion
	// config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	// clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	// rc := clientset.RESTClient()
	// // create a RESTClient

	// rc, err := rest.RESTClientFor(config)
	// if err != nil {
	// 	panic(err.Error())
	// }
	b := true
	opts := metav1.ListOptions{
		Watch:                true,
		SendInitialEvents:    &b,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}
	// attempts to begin watching the namespaces
	// returns a `watch.Interface`, or an error

	gvr, _ := schema.ParseResourceArg("pods.v1.")
	_, err = dyclient.Resource(*gvr).Watch(context.TODO(), opts)

	if err != nil {

		print(err.Error())
		panic(err)
	}
	watcher, err := dyclient.Resource(*gvr).Watch(context.TODO(), opts)

	// Create a stop channel to signal the goroutine to stop
	stopChannel := make(chan struct{})

	var wg sync.WaitGroup

	// Launch the goroutine and pass the channel as an argument
	wg.Add(1)
	go backgroundProcessor(watcher.ResultChan(), stopChannel, &wg)

	// Define the delay duration (e.g., 3 seconds)
	// delay := 15 * time.Second

	// time.Sleep(delay)

	// newCar := struct{}{}
	// stopChannel <- newCar
	// close(stopChannel)
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
			_, ok := <-stopCh
			// check if channel is closed
			if !ok {
				println("Goroutines killed!")
				wg.Done()
				return
			}
		}
	}
}

type registerWatchers struct {
	pipeline.StepContext
	dynamicKube dynamic.DynamicClient
	resource    string
}

func NewWatcherRegistrationStep(dynamicKube dynamic.DynamicClient, resource string) *registerWatchers {
	return &registerWatchers{
		dynamicKube: dynamicKube,
		resource:    resource,
	}
}

func (w registerWatchers) Exec(request *pipeline.Request) *pipeline.Result {

	b := true
	opts := metav1.ListOptions{
		Watch:                true,
		SendInitialEvents:    &b,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}
	// attempts to begin watching the namespaces
	// returns a `watch.Interface`, or an error

	gvr, _ := schema.ParseResourceArg(w.resource)
	watcher, err := w.dynamicKube.Resource(*gvr).Watch(context.TODO(), opts)
	return &pipeline.Result{
		Error: err,
		Data:  watcher,
	}
}

func (w registerWatchers) Cancel() error {
	return nil
}

type startWatchers struct {
	pipeline.StepContext
	stopChan    chan struct{}
	dynamicKube dynamic.DynamicClient
	resource    string
}

func newStartWatcherStage(dynamicKube dynamic.DynamicClient, resource string, stopChan chan struct{}) *startWatchers {
	return &startWatchers{
		stopChan:    stopChan,
		dynamicKube: dynamicKube,
		resource:    resource,
	}
}
func (w startWatchers) Exec(request *pipeline.Request) *pipeline.Result {

	b := true
	opts := metav1.ListOptions{
		Watch:                true,
		SendInitialEvents:    &b,
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}
	// attempts to begin watching the namespaces
	// returns a `watch.Interface`, or an error

	gvr, _ := schema.ParseResourceArg(w.resource)
	watcher, err := w.dynamicKube.Resource(*gvr).Watch(context.TODO(), opts)
	if err != nil {
		return &pipeline.Result{
			Error: err,
			Data:  nil,
		}
	}

	result := watcher.ResultChan()
	for {
		select {

		default:

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
			}
		}

		if len(w.stopChan) > 0 {
			break
		}

	}
	return &pipeline.Result{
		Error: nil,
		Data:  nil,
	}
}

func (w startWatchers) Cancel() error {
	return nil
}
