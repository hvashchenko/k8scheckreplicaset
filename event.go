package main

import (
	"fmt"
	"io/ioutil"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// creates the in-cluster config
	podName := os.Getenv("POD_NAME")
	dat, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		panic(err)
	}
	nsName := string(dat)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		pod, err := clientset.CoreV1().Pods(nsName).Get(podName, metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		if len(pod.OwnerReferences) < 1 {
			panic("pod has no owner references")
		}
		ownerRef := pod.OwnerReferences[0]

		switch ownerRef.Kind {
		case "ReplicaSet":
			WatchReplicaSet(clientset, nsName, ownerRef.Name)
		}

		time.Sleep(10 * time.Second)
	}
}

func WatchReplicaSet(clientset *kubernetes.Clientset, ns, name string) {
	selector, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	if err != nil {
		panic(err)
	}
	watchlist := cache.NewListWatchFromClient(clientset.ExtensionsV1beta1().RESTClient(), "replicasets", ns,
	selector)

	_, controller := cache.NewInformer(
		watchlist,
		&v1beta1.ReplicaSet{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("rs added: %s \n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				rs := obj.(*v1beta1.ReplicaSet)
				fmt.Printf("rs deleted: %s \n", rs.Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("rs changed \n")
			},
		},
	)
	stop := make(chan struct{})
	controller.Run(stop)

}

