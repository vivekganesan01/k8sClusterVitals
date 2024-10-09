package k8client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const statefulset = "statefulset"

func (wc *Watcher) checkStatefulsetHealth(statefulSet *v1.StatefulSet, initialDelaySeconds int16) {
	desiredReplicas := *statefulSet.Spec.Replicas
	time.Sleep(time.Duration(initialDelaySeconds) * time.Second)
	if statefulSet.Status.ReadyReplicas == desiredReplicas && statefulSet.Status.CurrentReplicas == desiredReplicas {
		log.Info().Str("caller", "check_statefulset_health").Str("tag", statefulset).Str("namespace", statefulSet.Namespace).Msg(helpers.LogMsg("statefulset is healthy: ", statefulSet.Name))
		wc.CacheStore.Delete(fmt.Sprintf("statefulset.apps/%s", statefulSet.Name))
	} else {
		UnavailableReplicas := desiredReplicas - statefulSet.Status.CurrentReplicas
		// todo: to reduce some work on cache, check for key existance first and set the cache
		wc.CacheStore.Set(fmt.Sprintf("statefulset.apps/%s", statefulSet.Name), []byte("unavailable"))
		log.Error().Str("caller", "check_statefulset_health").Str("tag", statefulset).Str("namespace", statefulSet.Namespace).Msg(helpers.LogMsg("statefulset is not healthy: ", statefulSet.Name, ", unavailable: ", strconv.FormatInt(int64(UnavailableReplicas), 10)))
	}
}

func (wc *Watcher) WatchStatefulSet(ctx context.Context, LabelSelector string) {
	defer wc.Wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("caller", "watch_statefulset").Str("tag", statefulset).Msg("gracefully shutting down statefulset watch")
			return
		default:
			time.Sleep(15 * time.Second) // todo param this timer
			statefulsets, err := wc.Clientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{
				LabelSelector: LabelSelector, // Use LabelSelector to filter statefulset by annotations
			})
			if err != nil {
				log.Info().Str("caller", "watch_statefulsets").Msg("no statefulset has been found...")
				continue
			}
			for _, statefulset := range statefulsets.Items {
				go func(sts v1.StatefulSet) {
					wc.checkStatefulsetHealth(&sts, 3)
				}(statefulset)
			}
		}
	}
}
