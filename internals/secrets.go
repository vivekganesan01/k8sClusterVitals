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

func (wc *Watcher) WatchSecrets(ctx context.Context) {
	defer wc.Wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("caller", "watch_secrets").Msg("gracefully shutting down secret watch")
			return
		default:
			time.Sleep(15 * time.Second) // todo param this timer
			data, err := wc.CacheStore.GoCacheGet("watch.secrets.config")
			if err != nil {
				log.Warn().Str("caller", "watch_secrets").Msg("watch.secrets.config cachehit doesnt exists")
				return
			}
			v := data.([]helpers.WatchedResource)
			// todo: move to go routine and make sure its completes
			for _, secretMetadata := range v {
				_, err := wc.Clientset.CoreV1().Secrets(secretMetadata.Namespace).Get(context.TODO(), secretMetadata.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					wc.CacheStore.Set(fmt.Sprintf("secrets.%s/%s", secretMetadata.Namespace, secretMetadata.Name), []byte("unavailable"))
					log.Info().Str("caller", "watch_secrets").Msg(helpers.LogMsg("Secret not found in namespace ", secretMetadata.Name, " namespace: ", secretMetadata.Namespace))
				} else if err != nil {
					wc.CacheStore.Set(fmt.Sprintf("secrets.%s/%s", secretMetadata.Namespace, secretMetadata.Name), []byte("invalid"))
					log.Error().Str("caller", "watch_secrets").Msg(helpers.LogMsg("error retrieving the secrets ", secretMetadata.Namespace, "-", secretMetadata.Name))
				} else {
					wc.CacheStore.Delete(fmt.Sprintf("secrets.%s/%s", secretMetadata.Namespace, secretMetadata.Name))
					log.Info().Str("caller", "watch_secrets").Msg(helpers.LogMsg("Secret found in namespace ", secretMetadata.Name, " namespace: ", secretMetadata.Namespace))
				}
			}
		}
	}
}
