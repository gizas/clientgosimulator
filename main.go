/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	_ "net/http/pprof"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Controller demonstrates how to implement a controller with client-go.
type Controller struct {
	indexer  cache.Store
	queue    workqueue.Interface
	informer cache.Controller
	resource string
}

// NewController creates a new Controller.
func NewController(queue workqueue.Interface, indexer cache.Indexer, informer cache.Controller, resource string) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		resource: resource,
	}
}

type item struct {
	object    interface{}
	objectRaw interface{}
	state     string
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	obj, quit := c.queue.Get()
	if quit {
		return false
	}

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(obj)

	//var entry *item
	//var ok bool
	//if entry, ok = obj.(*item); !ok {
	//	fmt.Println("\n\n\nCannot convert to item obj %v", obj)
	//	fmt.Println("Key is %v", obj.(string))
	//	return true
	//}

	//key, ok := entry.object.(string)
	//if !ok {
	//	return false
	//}

	// Invoke the method containing the business logic
	_ = c.syncToStdout(obj.(string))
	// Handle the error if something went wrong during the execution of the business logic
	//c.handleErr(err, obj)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the pod to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncToStdout(key string) error {
	resource := c.resource
	if resource == "pod" {
		o, exists, err := c.indexer.GetByKey(key)
		fmt.Println("\n\n\nCurrently watching key %v", o)
		if err != nil {
			return err
		}

		if !exists {
			// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
			fmt.Printf("%s %s does not exist anymore\n", resource, key)
		} else {
			// Note that you also have to check the uid if you have a local controlled resource, which
			// is dependent on the actual instance, to detect that a Pod was recreated with the same name
			// fmt.Printf("%s %s\n", resource, obj.(*v1.Pod).GetName())
			// for key, element := range obj.(*v1.Pod).GetLabels() {
			// 	fmt.Println("Pod:", obj.(*v1.Pod).GetName(), "Key:", key, "=>", "label:", element)
			// }
		}
	} else if resource == "namespace" {
		obj, exists, err := c.indexer.GetByKey(key)
		if err != nil {
			return err
		}

		if !exists {
			// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
			fmt.Printf("%s %s does not exist anymore\n", resource, key)
		} else {
			// Note that you also have to check the uid if you have a local controlled resource, which
			// is dependent on the actual instance, to detect that a Pod was recreated with the same name
			fmt.Printf("%s %s size:%d \n", resource, obj.(*v1.Namespace).GetName(), obj.(*v1.Namespace).Size())
		}
	} else if resource == "node" {
		obj, exists, err := c.indexer.GetByKey(key)
		if err != nil {
			return err
		}

		if !exists {
			// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
			fmt.Printf("%s %s does not exist anymore\n", resource, key)
		} else {
			// Note that you also have to check the uid if you have a local controlled resource, which
			// is dependent on the actual instance, to detect that a Pod was recreated with the same name
			fmt.Printf("%s %s\n", resource, obj.(*v1.Node).GetName())
		}
	}

	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
}

// Run begins watching and syncing.
func (c *Controller) Run(workers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// Server for pprof
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
	}

	// create the pod watcher
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", "", fields.Everything())
	namespaceListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "namespaces", "", fields.Everything())
	nodeListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "nodes", "", fields.Everything())

	//pods, err := podListWatcher.ListFunc(meta_v1.ListOptions{})
	//fmt.Printf("Size of Pods: %T, %d\n", pods, unsafe.Sizeof(pods))

	// create the workqueue
	queue := workqueue.NewNamed("pod")
	queuen := workqueue.NewNamed("ns")
	queueno := workqueue.NewNamed("node")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Pod than the version which was responsible for triggering the update.

	//POD
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})
	controller := NewController(queue, indexer, informer, "pod")

	//Namespace
	indexern, informern := cache.NewIndexerInformer(namespaceListWatcher, &v1.Namespace{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queuen.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queuen.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queuen.Add(key)
			}
		},
	}, cache.Indexers{})
	controllern := NewController(queuen, indexern, informern, "namespace")

	//Node
	indexerno, informerno := cache.NewIndexerInformer(nodeListWatcher, &v1.Node{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queueno.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queueno.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queueno.Add(key)
			}
		},
	}, cache.Indexers{})
	controllerno := NewController(queueno, indexerno, informerno, "node")

	// We can now warm up the cache for initial synchronization.
	// Let's suppose that we knew about a pod "mypod" on our last run, therefore add it to the cache.
	// If this pod is not there anymore, the controller will be notified about the removal after the
	// cache has synchronized.
	// indexer.Add(&v1.Pod{
	// 	ObjectMeta: meta_v1.ObjectMeta{
	// 		Name:      "mypod",
	// 		Namespace: v1.NamespaceDefault,
	// 	},
	// })

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)
	go controllern.Run(1, stop)
	go controllerno.Run(1, stop)

	// Wait forever
	select {}
}
