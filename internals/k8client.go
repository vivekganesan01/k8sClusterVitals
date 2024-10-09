package k8client

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Helper function to get the home directory
func homeDir() string {
	return os.Getenv("KUBE_HOME")
}

// const LabelSelector = "k8sclustervitals.io/scrape=true"

type Watcher struct {
	Clientset  *kubernetes.Clientset
	Queue      workqueue.RateLimitingInterface
	CacheStore *helpers.KeyValueStore
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

func (wc *Watcher) StartWatchingResources() {
	factory := informers.NewSharedInformerFactoryWithOptions(wc.Clientset, 120*time.Second, informers.WithNamespace("default"), informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
		lo.LabelSelector = "k8sclustervitals.io/scrape=true" // todo: move this to constant and remove default namespace
	}))

	// // Watch for Pods
	// podInformer := factory.Core().V1().Pods().Informer()
	// podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
	// 	AddFunc:    wc.onPodAdd,
	// 	UpdateFunc: wc.onPodUpdate,
	// 	DeleteFunc: wc.onPodDelete,
	// })

	// Watch for Deployments
	deployInformer := factory.Apps().V1().Deployments().Informer()
	deployInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    wc.onDeployAdd,
		UpdateFunc: wc.onDeployUpdate,
	})

	// Start all informers
	stopCh := make(chan struct{})
	defer close(stopCh)

	factory.Start(stopCh)
	// Wait for the informer caches to sync before acting on events
	if !cache.WaitForCacheSync(stopCh, deployInformer.HasSynced) {
		log.Error().Msg("Failed to sync informer caches")
		return
	}
	<-stopCh
}
