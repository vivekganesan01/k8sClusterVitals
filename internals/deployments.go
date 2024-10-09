package k8client

import (
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"

	v1 "k8s.io/api/apps/v1"
)

const deployments = "deployment"

func checkDeploymentHealth(deploy *v1.Deployment, cache *helpers.KeyValueStore, initialDelaySeconds int16) {
	// log.Info().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("checking deployment status for ", deploy.Name, " under ", deploy.Namespace))
	time.Sleep(time.Duration(initialDelaySeconds) * time.Second)
	if deploy.Status.AvailableReplicas == *deploy.Spec.Replicas && deploy.Status.UnavailableReplicas == 0 {
		log.Info().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("deployment is healthy: ", deploy.Name))
		cache.Delete(deploy.Name)
	} else {
		// todo: to reduce some work on cache, check for key existance first and set the cache
		cache.Set(deploy.Name, []byte("unavailable"))
		log.Error().Str("caller", "check_deployment_health").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("deployment is not healthy: ", deploy.Name, ", unavailable: ", strconv.FormatInt(int64(deploy.Status.UnavailableReplicas), 10)))
	}
}

func (wc *Watcher) onDeployAdd(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	log.Info().Str("caller", "on_deploy_add").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("new deployment added: ", deploy.Name))
	checkDeploymentHealth(deploy, wc.CacheStore, 20)
}

func (wc *Watcher) onDeployUpdate(oldObj, newObj interface{}) {
	deploy := newObj.(*v1.Deployment)
	log.Info().Str("caller", "on_deploy_update").Str("tag", deployments).Str("namespace", deploy.Namespace).Msg(helpers.LogMsg("syncing deployment updates: ", deploy.Name))
	checkDeploymentHealth(deploy, wc.CacheStore, 10)
}
