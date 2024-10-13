package k8client

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
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

var scrapeConfig helpers.ScrapeConfiguration

type Watcher struct {
	Clientset  *kubernetes.Clientset
	Queue      workqueue.RateLimitingInterface
	CacheStore *helpers.KeyValueStore
	Wg         sync.WaitGroup
}

func (wc *Watcher) syncScrapeConfiguration(configMap *corev1.ConfigMap, reason string) {
	if reason == "delete" {
		wc.CacheStore.GoCacheDelete("watch.secrets.config")
		wc.CacheStore.GoCacheDelete("watch.configmaps.config")
		return
	}
	// need this to refresh cache upon scrape configuration update
	wc.CacheStore.GoCacheDelete("watch.secrets.config")
	wc.CacheStore.GoCacheDelete("watch.configmaps.config")

	watched_secrets, ok := configMap.Data["watched-secrets"]
	if !ok {
		log.Info().Str("caller", "sync_scrape_configuration").Msg("watched-secrets not found in the scrape configuration")
		wc.CacheStore.GoCacheDelete("watch.secrets.config")
	} else {
		yaml.Unmarshal([]byte(watched_secrets), &scrapeConfig.WatchedSecrets)
		log.Info().Str("caller", "sync_scrape_configuration").Msg(helpers.LogMsg("watched-secrets set for event ", reason))
		wc.CacheStore.GoCacheSet("watch.secrets.config", scrapeConfig.WatchedSecrets)
	}

	watched_configmaps, ok := configMap.Data["watched-configmaps"]
	if !ok {
		log.Info().Str("caller", "sync_scrape_configuration").Msg("watched-configmaps not found in the scrape configuration")
		wc.CacheStore.GoCacheDelete("watch.secrets.config")
	} else {
		yaml.Unmarshal([]byte(watched_configmaps), &scrapeConfig.WatchedConfigMaps)
		log.Info().Str("caller", "sync_scrape_configuration").Msg(helpers.LogMsg("watched-configmap set for event ", reason))
		wc.CacheStore.GoCacheSet("watch.configmaps.config", scrapeConfig.WatchedConfigMaps)
	}
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

// WatchConfigMap watches the config map for changes and updates the secret watcher
func (wc *Watcher) WatchScrapeConfig() {
	log.Info().Str("caller", "watch_scrape_config").Msg("mointoring scrape config yaml file")
	informerFactory := informers.NewSharedInformerFactoryWithOptions(wc.Clientset, 30*time.Second,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = "k8sclustervitals.io/config=exists"
		}),
	)
	configMapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
	// Add an event handler to the informer to react to changes
	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*corev1.ConfigMap)
			log.Info().Str("caller", "watch_scraper_config_add_event").Msg(helpers.LogMsg("identified scraper config ", configMap.Namespace, " ", configMap.Name))
			wc.syncScrapeConfiguration(configMap, "added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newConfigMap := newObj.(*corev1.ConfigMap)
			log.Info().Str("caller", "watch_scraper_config_update_event").Msg(helpers.LogMsg("identified update scraper config ", newConfigMap.Namespace, " ", newConfigMap.Name))
			wc.syncScrapeConfiguration(newConfigMap, "updated")
		},
		DeleteFunc: func(obj interface{}) {
			configMap := obj.(*corev1.ConfigMap)
			log.Info().Str("caller", "watch_scraper_config_delete_event").Msg(helpers.LogMsg("identified delete scraper config ", configMap.Namespace, " ", configMap.Name))
			wc.syncScrapeConfiguration(configMap, "delete")
		},
	})

	// Start the informer
	stopCh := make(chan struct{})
	defer close(stopCh)
	go configMapInformer.Run(stopCh)
	// Wait for the caches to sync
	if !cache.WaitForCacheSync(stopCh, configMapInformer.HasSynced) {
		log.Info().Str("caller", "watch_scrape_config").Msg("timrf out waiting for caches to sync")
		return
	}
	// Keep the program running
	select {}
	// Inform the waitgroup that we're starting a new goroutine
	// go func() {
	// 	// defer wc.Wg.Done() // Ensure that the waitgroup is decremented when the informer stops
	// 	stopCh := make(chan struct{})
	// 	defer close(stopCh)
	// 	informerFactory.Start(stopCh)
	// 	informerFactory.WaitForCacheSync(stopCh)
	// }()
	// todo: check for kill signals
	// wc.Wg.Wait()
}

// func (wc *Watcher) checkScrapeConfig(configMap *corev1.ConfigMap, reason string) {

// 	if reason == "delete" {
// 		wc.CacheStore.GoCacheDelete("watch.secrets.config")
// 		wc.CacheStore.GoCacheDelete("watch.configmaps.config")
// 		return
// 	}

// 	var err error
// 	log.Info().Str("caller", "checkScrapeConfig").Msg(helpers.LogMsg("identified scrape configuration ", configMap.Namespace, " ", configMap.Name))
// 	var scrapeConfig helpers.ScrapeConfiguration
// 	// for secret
// 	err = yaml.Unmarshal([]byte(configMap.Data["watched-secrets"]), &scrapeConfig.WatchedSecrets)
// 	if err != nil {
// 		log.Error().Str("caller", "check_scrape_config").Msg(helpers.LogMsg("found invalid scrape configuration", configMap.Namespace, configMap.Name))
// 	} else {
// 		// todo: clean up this code
// 		_, err := json.Marshal(scrapeConfig)
// 		if err != nil {
// 			log.Error().Str("caller", "check_scrape_config").Msg(helpers.LogMsg("failed to cache the scrape configuration", configMap.Namespace, configMap.Name))
// 		} else {
// 			log.Info().Str("caller", "check_scrape_config").Msg("updating scrape config to cache - secret")
// 			wc.CacheStore.GoCacheSet("watch.secrets.config", scrapeConfig.WatchedSecrets)
// 		}
// 	}
// 	// for cm
// 	err = yaml.Unmarshal([]byte(configMap.Data["watched-configmaps"]), &scrapeConfig.WatchedConfigMaps)
// 	if err != nil {
// 		log.Error().Str("caller", "check_scrape_config").Msg(helpers.LogMsg("found invalid scrape configuration for cm", configMap.Namespace, configMap.Name))
// 	} else {
// 		_, err := json.Marshal(scrapeConfig)
// 		if err != nil {
// 			log.Error().Str("caller", "check_scrape_config").Msg(helpers.LogMsg("found to cache the scrape configuration", configMap.Namespace, configMap.Name))
// 		} else {
// 			log.Info().Str("caller", "check_scrape_config").Msg("updating scrape config to cache - cm")
// 			wc.CacheStore.GoCacheSet("watch.configmaps.config", scrapeConfig.WatchedConfigMaps)
// 		}
// 	}
// }

func (wc *Watcher) StartWatchingResources(ctx context.Context, LabelSelector string) {
	// wc.Wg.Add(1)
	go wc.WatchScrapeConfig()
	wc.Wg.Add(1)
	go wc.WatchDeployment(ctx, LabelSelector)
	wc.Wg.Add(1)
	go wc.WatchStatefulSet(ctx, LabelSelector)
	wc.Wg.Add(1)
	go wc.WatchSecrets(ctx)
	wc.Wg.Add(1)
	go wc.WatchConfigMaps(ctx)
}
