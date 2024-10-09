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

const deployments = "deployment"

func (wc *Watcher) checkDeploymentHealth(deploy *v1.Deployment, initialDelaySeconds int16) {
	// log.Info().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("checking deployment status for ", deploy.Name, " under ", deploy.Namespace))
	time.Sleep(time.Duration(initialDelaySeconds) * time.Second)
	if deploy.Status.AvailableReplicas == *deploy.Spec.Replicas && deploy.Status.UnavailableReplicas == 0 {
		log.Info().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("deployment is healthy: ", deploy.Name))
		wc.CacheStore.Delete(fmt.Sprintf("deployment.apps/%s", deploy.Name))
	} else {
		// todo: to reduce some work on cache, check for key existance first and set the cache
		wc.CacheStore.Set(fmt.Sprintf("deployment.apps/%s", deploy.Name), []byte("unavailable"))
		log.Error().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("deployment is not healthy: ", deploy.Name, ", unavailable: ", strconv.FormatInt(int64(deploy.Status.UnavailableReplicas), 10)))
	}
}

func (wc *Watcher) WatchDeployment(ctx context.Context, LabelSelector string) {
	defer wc.Wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("caller", "watch_deployment").Msg("gracefully shutting down deployment watch")
			return
		default:
			time.Sleep(15 * time.Second) // todo param this timer
			deployments, err := wc.Clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{
				LabelSelector: LabelSelector, // Use LabelSelector to filter deployments by annotations
			})
			if err != nil {
				log.Info().Str("caller", "watch_deployment").Msg("no deployment has been found...")
				continue
			}
			for _, deployment := range deployments.Items {
				// todo: may be need to add context here as well ?
				go func(deployment v1.Deployment) {
					wc.checkDeploymentHealth(&deployment, 3)
				}(deployment)
			}
		}
	}
}
