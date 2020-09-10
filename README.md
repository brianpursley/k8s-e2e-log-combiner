# k8s-e2e-log-combiner
Combines all log artifacts into a single file, sorted by timestamp.

It takes one argument, which can be either:
1. A `prow.k8s.io` url (or the `gcsweb.k8s.io` url). 
2. A local directory containing log files

It will read `build-log.txt` and all files ending in `.log`, combine them, sort them, and write the output to stdout.

# Usage Examples (Go Run)

### Combine log files from a prow url:
```
go run combiner.go https://prow.k8s.io/view/gcs/kubernetes-jenkins/pr-logs/pull/92064/pull-kubernetes-e2e-gce-ubuntu-containerd/1301618335330340866
```

### Combine log files from a local path:
```
go run combiner.go /path/to/log/files
```

# Usage Examples (Docker)

### Combine log files from a prow url:
```
docker run brianpursley/k8s-e2e-log-combiner https://prow.k8s.io/view/gcs/kubernetes-jenkins/pr-logs/pull/92064/pull-kubernetes-e2e-gce-ubuntu-containerd/1301618335330340866
```

### Combine log files from a local path:
For this you will need to mount your local directory into the docker container, like this:
```
docker run -v /path/to/log/files:/foo brianpursley/k8s-e2e-log-combiner /foo 
```

# Output format
Output consists of a timestamp, a truncated filename, and the log message.

Example excerpt:
```
20:49:58.096058452 [/artifacts/e2e-40...c-a7d53-minion-group-hfgh/containerd.log] time="2020-09-03T20:49:58.096058452Z" level=info msg="Exec process \"4433fe624c90bbc45c4943bf1986f73286dcd8707ca4f8ebcf569b28dd76896a\" exits with exit code 0 and error <nil>"
20:49:58.100716000 [/artifacts/e2e-40...135c-a7d53-minion-group-hfgh/kubelet.log] I0903 20:49:58.100716    9097 desired_state_of_world_populator.go:361] Added volume "default-token-ngcz4" (volSpec="default-token-ngcz4") for pod "df47b3c8-a32a-4cd2-8ae2-c84d4be52016" to desired state.
20:49:58.100895000 [/artifacts/e2e-40...135c-a7d53-minion-group-hfgh/kubelet.log] I0903 20:49:58.100895    9097 shared_informer.go:270] caches populated
20:49:58.100954000 [/artifacts/e2e-40...135c-a7d53-minion-group-hfgh/kubelet.log] I0903 20:49:58.100954    9097 shared_informer.go:270] caches populated
20:49:58.105874881 [/artifacts/e2e-40...c-a7d53-minion-group-hfgh/containerd.log] time="2020-09-03T20:49:58.105874881Z" level=info msg="Finish piping \"stdout\" of container exec \"4433fe624c90bbc45c4943bf1986f73286dcd8707ca4f8ebcf569b28dd76896a\""
20:49:58.106509345 [/artifacts/e2e-40...c-a7d53-minion-group-hfgh/containerd.log] time="2020-09-03T20:49:58.106509345Z" level=info msg="Finish piping \"stderr\" of container exec \"4433fe624c90bbc45c4943bf1986f73286dcd8707ca4f8ebcf569b28dd76896a\""
20:49:58.126818000 [/artifacts/e2e-40...135c-a7d53-minion-group-hfgh/kubelet.log] I0903 20:49:58.126818    9097 reconciler.go:254] Starting operationExecutor.MountVolume for volume "default-token-ngcz4" (UniqueName: "kubernetes.io/secret/df47b3c8-a32a-4cd2-8ae2-c84d4be52016-default-token-ngcz4") pod "affinity-nodeport-timeout-ww59w" (UID: "df47b3c8-a32a-4cd2-8ae2-c84d4be52016") Volume is already mounted to pod, but remount was requested.
```