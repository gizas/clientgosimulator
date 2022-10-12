# Elastic Agent Simulator Example

This example demonstrates how to write a controller which follows the states
of watched resources in a k8s cluster


## Building
```
docker build -t in-cluster17_vendor .
docker tag in-cluster17_vendor:latest andreasgkizas/2159_17:v5
```

> Dockerfile is part of local repositiry. Assumes `RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o clientgo` image target is linux based.

## Uploading to 
```Upload to a GKE artifactory
docker tag in-cluster17_vendor:latest andreasgkizas/2159_17:v5
docker push andreasgkizas/2159_17:v5
```

## Update the Manifest accordingly to use inside your cluster

See worker_manifestv3_workingGKE.yaml.txt as working one provided to customer
Also serviceAccountName: filebeat is used inside, assumes that filebeats service account is already present. See [here](https://github.com/elastic/beats/blob/main/deploy/kubernetes/filebeat-kubernetes.yaml#L135) 
