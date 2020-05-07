# Website Operator

A Kubernetes operator that helps deploy static websites on a Kubernetes cluster.

## Static files

Static files are placed on a bucket:

```shell
$ gsutil mb gs://$PROJECT
Creating gs://website-operator/...
$ gsutil iam ch allUsers:objectViewer gs://$PROJECT
$ gsutil cp -R public/ gs://$PROJECT/
[...]
Operation completed over 172 objects/9.1 MiB.
```

## First step: manual deployment

```shell
$ kubectl create namespace manual
namespace/manual created

$ kubectl apply -f manual.yaml

$ IP=$(kubectl get svc website -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# verify pods scale as expected
$ ab ab -n 10000 -c 200 http://$IP
```

## Next step: create an operator

Install kubebuilder from https://github.com/kubernetes-sigs/kubebuilder/releases

> Explain GVK (Group, Version, Kind)

Initialize the project:

```shell
$ go mod init example.com/website/v1alpha1

$ kubebuilder init --domain example.com

$ kubebuilder create api --group website --version v1alpha1 --kind Static
```

Deploy the definition of the custom resource (CRD):

```shell
# Declare the custom resource to the API
$ make install
customresourcedefinition.apiextensions.k8s.io/statics.website.example.com created

# Create a new instance of the resource
$ kubectl apply -f config/samples/website_v1alpha1_static.yaml
static.website.example.com/static-sample created

# Get instances
$ kubectl get statics.website.example.com  
NAME            AGE
static-sample   2s
```

> Explain reconciliation loop

```shell
# Run the operator locally (with user rights)
$ make run
[...]
DEBUG   controller-runtime.controller Successfully Reconciled {"controller": "static", "request": "operated/static-sample"}

# On another terminal
$ kubectl delete statics.website.example.com static-sample

# On first terminal
DEBUG   controller-runtime.controller Successfully Reconciled {"controller": "static", "request": "operated/static-sample"}
```


```shell
# Build operator image and push it to registry
$ IMG=eu.gcr.io/$PROJECT/operator:1 make docker-build docker-push

# Deploy the operator to the Kubernetes cluster with specific rights
$ IMG=eu.gcr.io/$PROJECT/operator:1 make deploy
namespace/operator-system created
customresourcedefinition.apiextensions.k8s.io/statics.website.example.com configured
role.rbac.authorization.k8s.io/operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/operator-manager-role created
clusterrole.rbac.authorization.k8s.io/operator-proxy-role created
clusterrole.rbac.authorization.k8s.io/operator-metrics-reader created
rolebinding.rbac.authorization.k8s.io/operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/operator-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/operator-proxy-rolebinding created
service/operator-controller-manager-metrics-service created
deployment.apps/operator-controller-manager created

$ kubectl get pods -n operator-system
NAME                                           READY   STATUS    RESTARTS   AGE
operator-controller-manager-767676dbd5-rpw6w   2/2     Running   0          11s
```

### Specify fields in the resource

- `diskSize`: the size to reserve for the assets
- `source`: source of the assets, in the form gs://bucket-name/path
- `minReplicas`: Min replicas of webserver instances
- `maxReplicas`: Max replicas of webserver instances

> See `operator/api/v1alpha1/static_types.go:28-40`

```go
// StaticSpec defines the desired state of Static
type StaticSpec struct {
  // DiskSize indicates the amount of disk space to reserve to store assets for each instance
  DiskSize string `json:"diskSize"`

  // Source indicates the source of the assets to serve, in the form `gs://bucket-name/path`
  Source string `json:"source"`

  // MinReplicas indicates the minimal number of instances to deploy
  MinReplicas int `json:"minReplicas"`

  // MaxReplicas indicates the maximal number of instances to deploy
  MaxReplicas int `json:"maxReplicas"`
}
```

> See `operator/controllers/static_controller.go`

```go
static := new(websitev1alpha1.Static)
if err := r.Get(ctx, req.NamespacedName, static); err != nil {
  // we'll ignore not-found errors, since they can't be fixed by an immediate
  // requeue (we'll need to wait for a new notification), and we can get them
  // on deleted requests.
  err = client.IgnoreNotFound(err)
  return ctrl.Result{}, err
}
log.Info(fmt.Sprintf("static: %+v", static.Spec))
```

Modify the resource sample:

> See `operator/config/samples/website_v1alpha1_static.yaml`

```yaml
apiVersion: website.example.com/v1alpha1
kind: Static
metadata:
  name: static-sample
spec:
  diskSize: 20M
  source: gs://website-operator/public
  minReplicas: 1
  maxReplicas: 4
```

Update the sample:

```shell
$ kubectl apply -f config/samples/website_v1alpha1_static.yaml
static.website.example.com/static-sample configured
```

Build and deploy the new version:

```shell
$ IMG=eu.gcr.io/$PROJECT/operator:2 make docker-build docker-push deploy
```
