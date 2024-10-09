package k8client

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Helper function to get the home directory
func homeDir() string {
	return os.Getenv("KUBE_HOME")
}

type Watcher struct {
	Clientset  *kubernetes.Clientset
	Queue      workqueue.RateLimitingInterface
	CacheStore *helpers.KeyValueStore
	Wg         sync.WaitGroup
}

func NewKubeClient(cache *helpers.KeyValueStore) (*Watcher, error) {
	env := os.Getenv("ENV")
	var err error
	var config *rest.Config
	var kubeconfig *string
	// Get the kubeconfig path from the user's home directory.
	if env == "local" {
		log.Info().Str("func", "NewKubeClient").Msg("setting up k8client via ./kube/config ... starting....")
		if home := homeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	} else if env == "production" {
		log.Info().Str("func", "NewKubeClient").Msg("setting up k8client via service token ... starting....")
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid runtime")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	watcher := &Watcher{
		Clientset:  clientset,
		Queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		CacheStore: cache,
	}
	return watcher, nil
}

func (wc *Watcher) StartWatchingResources(ctx context.Context, LabelSelector string) {
	wc.Wg.Add(1)
	go wc.WatchDeployment(ctx, LabelSelector)
	wc.Wg.Add(1)
	go wc.WatchStatefulSet(ctx, LabelSelector)
}
