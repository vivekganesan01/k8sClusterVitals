package k8client

import (
	"fmt"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/apps/v1"
)

func (wc *Watcher) onStatefulSetAdd(obj interface{}) {
	statefulSet := obj.(*v1.StatefulSet)
	fmt.Printf("StatefulSet added: %s\n", statefulSet.Name)

	if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
		log.Info().Msg("StatefulSet is healthy")
	} else {
		log.Warn().Msg("StatefulSet is not healthy")
	}
}
