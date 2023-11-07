# streaming-lists
Golang Tutorial to show implementation of streaming lists

## Introduction 
From Kubernetes 1.27 a feature WatchList was introduced to stem potential runnaway Memory usage 
due to LIST command. Traditionally the LIST command chunks the events collected on a particular
Kubernetes object and sends them as such. This has a potential for hogging memory in a cluster that may
have adverse consequences including stopping the actual kubelet. 
This command is used whenever one wants to watch for resource, the WATCH command has a LIST command underneath 
to send the initial events then a WATCH command to record subsequent events. 

With this feature one circumvents the LIST command, and instead directly uses the WATCH command that will stream the 
required information, also served from a Cache to further insulate the Memory from being drained. 

This implementation shows how to use this feature, which is still in the alpha stage and so one has to make sure the cluster has
this feature gate enabled.


### Sample
The sample program shows how we can use the streaming list to watch the events for a pod. It also uses the dynamic kubernetes client 
for easy lift and shift to any other resources.

This [script](./script.sh) also serves as a helper function that creates an instance of a pod using nginx image then deletes to generate events