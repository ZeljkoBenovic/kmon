# KMON  
Kubernetes administrators swiss knife. Kmon will automate boring, frequently performed tasks in Kubernetes.   
Some examples: 
* create a pod using a specified PVC to check its content
* create a pod with a PVC restored from a specific `VolumeSnapshot`
* create a PVC from `VolumeSnapshot`
* and the list goes on...

Kmon can be used as a standalone CLI tool, like any other. 
But its true power comes from integration with [k9s](https://github.com/derailed/k9s) as a [k9s plugin](https://k9scli.io/topics/plugins/)

## Usage
### Standalone CLI tool
* POD modes `kmon pod`:
    * Create a pod from PVC `--mode run-from-pvc`
    * Create a pod with a PVC restored from VolumeSnapshot `--mode run-from-snapshot`
      * `--mount-path string`      mount path (default "kmon-mnt")
      * `--name string`            pod name (default "kmon-pod")
      * `--pvc-name string`        pvc name (default "kmon-pvc")
      * `--snapshot-name string`   snapshot name (default "kmon-snapshot")
      * `--volume-name string`     volume name (default "kmon-volume")
      * `--context string`         kubeconfig context to use
      * `-n, --namespace string`   namespace to run in (default "default")
* PVC modes `kmon pvc`
  * Create VolumeSnapshot from PVC `--mode snapshot-from-pvc`
    * `--name string`                  pvc name (default "kmon-pvc")
    * `--snapshot-class-name string`   snapshot class name
    *  `--snapshot-name string`        snapshot name (default "kmon-snap")
    *  `--source-pvc-name string`      source pvc name

### K9s plugin
To configure `kmon` as a `k9s` plugin, check out [k9s-plugin.yaml](examples/k9s-plugin.yaml) for reference

### K8s CronJob
`kmon` potentially can be used to automate repetitive tasks, like creating `VolumeSnapshots` on a schedule using `CronJob` for example. 

## TBD
* Import AWS Volume Snapshot into K8s `VolumeSnapshot` - for situations when we're copying volumes across regions for example.
* Delete the existing PVC and replace it with another one with the same name, using `VolumeSnapsthot` to restore data and restart workloads.
* Discover pods and send http requests to all of them at once - for troubleshooting data propagation on distributed systems for example.
* If there is anything else you think it would be useful, feel free to create an issue with a feature request or create a PR. 