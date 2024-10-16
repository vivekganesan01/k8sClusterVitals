# k8sClusterVitals

k8sClusterVitals is an open-source health status monitoring app designed to provide real-time insights into the health of your Kubernetes cluster. It continuously monitors critical Kubernetes resources such as Deployments, StatefulSets, DaemonSets, Secrets, and ConfigMaps, ensuring smooth operation by detecting and reporting unhealthy objects.

If any resources become unhealthy, k8sClusterVitals will immediately report the issue through an exposed API endpoint, allowing you to take quick corrective action.

With the exposed API endpoint, you can use it to trace or monitor vital components and take action, such as shifting traffic in the load balancer, based on these health metrics.

## Features
- **Continuous Monitoring**: Constantly monitors key Kubernetes resources.
- **Real-Time Health Reports**: Detects unhealthy objects and reports them via an API.
- **Resource Coverage**: Monitors Deployments, StatefulSets, DaemonSets, Secrets, and ConfigMaps.
- **Exposed REST API**: Provides a REST API endpoint that can be used to monitor unhealthy services and the overall status of Kubernetes resources.
- **Label-Based Monitoring**: Monitors the health of Deployments and StatefulSets based on labels.
- **ConfigMap and Secret Monitoring**: Secrets and ConfigMaps are tracked based on user-defined inputs.

## Usage
k8sClusterVitals can monitor the following Kubernetes resources:

- Deployments
- StatefulSets
- Secrets
- ConfigMaps

For ***Deployments and StatefulSets***:
======================================
To monitor the health of the resource, you must add a label called **k8sclustervitals.io/scrape=true**. Once labeled, these resources will be actively monitored by k8sClusterVitals. If the health of a labeled Deployment or StatefulSet is compromised, it will be reported via the API endpoint.

Here’s an example of how to label a Deployment for monitoring:

eg: refer [sample_deployment.yaml](./examples/sample_deployment.yaml)

eg: refer [sample_statefulset.yaml](./examples/sample_statefulset.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
    k8sclustervitals.io/scrape: "true" # Important: This label is required for monitoring
spec:
  replicas: 6
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              memory: "128Mi"
              cpu: "250m"
            limits:
              memory: "256Mi"
              cpu: "500m"

```
## Example Output:
#### If a resource is unhealthy, the following command will return not_ok:
---
```
curl -X GET http://localhost:1323/healthcheck/v1/health
# returns not_ok
```
For continuous status updates:
```
while true; do curl http://localhost:1323/healthcheck/v1/status; sleep 1; done
```
The output will show:
```
{"deployment.apps/nginx-deployment": "Unavailable"}
```
#### When the service is healthy, the API will return ok and a blank status:
```
curl -X GET http://localhost:1323/healthcheck/v1/health
# returns ok
```
Continuous status check:
```
while true; do curl http://localhost:1323/healthcheck/v1/status; sleep 1; done
```
Output:
```
{}
```

For ***Secrets and ConfigMaps***:
=================================

k8sClusterVitals tracks whether a specified Secret or ConfigMap exists. Users need to provide the names of the Secrets and ConfigMaps to be watched via a ConfigMap, and those resources will be monitored accordingly.

To track a Secret or ConfigMap, you can optionally configure the following information:

eg: refer sample scrape configuration file [watcher_configmap.yaml](./examples/watcher_configmap.yaml)

```yaml
watched-secrets: |        # optional
  - name: my-secret       # name of the secret to watch
    namespace: default    # namespace where the secret resides
watched-configmaps: |     # optional
  - name: my-cm           # name of the configmap to watch
    namespace: default    # namespace where the configmap resides
```

#### Important note:
- The ConfigMap should be located in the same namespace where the k8sClusterVitals Deployment exists.
- The ConfigMap must include the label `k8sclustervitals.io/config=exists`.
- The ConfigMap's name doesn't matter, as it uses the label to identify the configuration.

Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: secret-cm-watcher-config
  labels:
    k8sclustervitals.io/config: "exists" # Important: this label should exists for monitoring
data:
  watched-secrets: |
    - name: my-secret
      namespace: default
  watched-configmaps: |
    - name: my-cm
      namespace: default
```
This setup will allow k8sClusterVitals to monitor the specified Secrets and ConfigMaps based on the provided configuration.

## Example Output:
#### If a provided secret or configmap does not exists, the following command will return not_ok:
---
```
curl -X GET http://localhost:1323/healthcheck/v1/health
# return not_ok
```
For continuous status updates:
```
while true; do curl http://localhost:1323/healthcheck/v1/status; sleep 1; done
```
The output will show:
```
{"secrets.default/my-secret": "Unavailable"}
```
### When the secret or configmap is available, the API will return ok and a blank status:
```
curl -X GET http://localhost:1323/healthcheck/v1/health
# returns ok
```
Continuous status check:
```
while true; do curl http://localhost:1323/healthcheck/v1/status; sleep 1; done
```
Output:
```
{}
```


> You can monitor any number of Deployments, StatefulSets, Secrets, and ConfigMaps. k8sClusterVitals will continuously track these resources and report their status via the exposed API endpoint, ensuring that you receive real-time health updates and can take necessary action. The system also supports retries for tracking in case of initial failure.

---
## Installation:

k8sClusterVitals supports both local and cluster deployments.

***Via local deployment:***

```bash
# Note: the following two environment variables are required
# Step 1: Export this variable:

export KUBE_HOME=${HOME}
export ENV="kubeconfig"

# Step 2: To run the code:

#  - Navigate to the bin directory.
#  - Identify the appropriate binary for your operating system:
#      - linux64-amd
#      - linux64-arm
#      - darwin64-amd
#      - darwin64-arm
#  - Run the corresponding binary file:

# eg: For macOS

./bin/k8sclustervitals-v0.0.1-darwin-amd6

```

***Via Docker Local:***

```bash
docker build -t k8sclustervitals -f Dockerfile.production ${pwd}
docker run -v ${HOME}/.kube:/root/.kube -e KUBE_HOME="/home" -e ENV="kubeconfig" -p 1323:1323  k8sclustervitals:latest
```

***Via DockerHub:***

```bash
docker pull k8sclustervitals:latest
docker run -v ${HOME}/.kube:/home/.kube -e KUBE_HOME="/home" -e ENV="kubeconfig" -p 1323:1323 k8sclustervitals:latest
```

***Via Cluster Deployment:***


```bash
# For cluster deployments, pass the required environment variable to the pod via the Deployment YAML:

# Step 1: Navigate to the Helm chart directory

    cd ./charts
 
# Step 2: Install the Helm chart

# Run the following command to install the `k8clustervitals` chart. You can customize the release name (`my-release`) as needed. Note: namespace has to be `k8cv`

   helm install my-release ./k8sclustervitals --create-namespace --namespace k8cv

 # - `my-release`: This is the release name. You can choose any name for the release.
 # - `./k8clustervitals`: This is the relative path to the Helm chart.
 # - `--namespace`: for now it must be "k8cv"

 #  Example:
helm install my-release ./k8sclustervitals  --create-namespace --namespace k8cv
```
