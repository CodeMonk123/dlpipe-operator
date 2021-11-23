package k8sutil

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewKubeClient() kubernetes.Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientSet := kubernetes.NewForConfigOrDie(config)
	return clientSet
}
