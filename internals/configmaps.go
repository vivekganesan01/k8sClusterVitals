package k8client

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// watch for resource under my configmap map cache
// alert the cache

func (wc *Watcher) WatchConfigMaps(ctx context.Context) {
	defer wc.Wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("caller", "watch_configmaps").Msg("gracefully shutting down configmap watch")
			return
		default:
			time.Sleep(15 * time.Second) // todo param this timer
			data, err := wc.CacheStore.GoCacheGet("watch.configmaps.config")
			if err != nil {
				log.Warn().Str("caller", "watch_configmaps").Msg("watch.configmaps.config cachehit doesnt exists")
				return
			}
			v := data.([]helpers.WatchedResource)
			// todo: move to go routine and make sure its completes
			for _, cmMetadata := range v {
				_, err := wc.Clientset.CoreV1().ConfigMaps(cmMetadata.Namespace).Get(context.TODO(), cmMetadata.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					wc.CacheStore.Set(fmt.Sprintf("configmaps.%s/%s", cmMetadata.Namespace, cmMetadata.Name), []byte("unavailable"))
					log.Info().Str("caller", "watch_configmaps").Msg(helpers.LogMsg("configmap not found in namespace ", cmMetadata.Name, " namespace: ", cmMetadata.Namespace))
				} else if err != nil {
					wc.CacheStore.Set(fmt.Sprintf("configmaps.%s/%s", cmMetadata.Namespace, cmMetadata.Name), []byte("invalid"))
					log.Error().Str("caller", "watch_configmaps").Msg(helpers.LogMsg("error retrieving the configmap ", cmMetadata.Namespace, "-", cmMetadata.Name))
				} else {
					wc.CacheStore.Delete(fmt.Sprintf("configmaps.%s/%s", cmMetadata.Namespace, cmMetadata.Name))
					log.Info().Str("caller", "watch_configmaps").Msg(helpers.LogMsg("configmap found in namespace ", cmMetadata.Name, " namespace: ", cmMetadata.Namespace))
				}
			}
		}
	}
}
