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

- Install kubebuilder from https://github.com/kubernetes-sigs/kubebuilder/releases

- Initialize the project:

  ```shell
  $ go mod init example.com/website/v1alpha1

  $ kubebuilder init --domain example.com
  
  $ kubebuilder create api --group website --version v1alpha1 --kind Static
  ```
