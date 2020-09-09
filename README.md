# k8s-e2e-log-combiner
Combines all log artifacts into a single file, sorted by timestamp.

It takes one argument: the `prow.k8s.io` url (or the `gcsweb.k8s.io` url). It will read `build-log.txt` and all file ending in `.log`, combine them, sort them, and write the output to stdout.

# Local Usage Example
```
go run combiner.go https://prow.k8s.io/view/gcs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-new-master-gci-kubectl-skew/1300673402025021440
```

# Docker Usage Example
```
docker run brianpursley/k8s-e2e-log-combiner https://prow.k8s.io/view/gcs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-new-master-gci-kubectl-skew/1300673402025021440
```
