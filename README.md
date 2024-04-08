## k8s based Router demo

This repo contains the prototype of the system routing requests to the backend based on hash value of the destination. This routing could be overwritten by rules stored in k8s config map.
The application has two components - StateFull set of pods (backend), the router pod. The router has two containers - nginx which terminates SSL and then forwards data to local port. Another container, router itself, listens on this port. The router forwards data to the backend pod. It selects right destination using simple hashing of namespace (parameter of URL). On the top of the hashing there is a table keeping overwriting rules. These rules are stored in ConfigMap. The router listens for changes of this resource nad every time the mapping gets changed the router updates
routing logic.

### Setup local registry
To run it locally you need to install minikube and docker.
You need a local registry to push images to k8s. minikube allows to use insecure registry:

`minikube start --insecure-registry "10.0.0.0/24"`

Enable the registry addon to allow Docker to push images to minikube's registry:

`minikube addons enable registry`

In a separate terminal, redirect port 5000 from Docker to port 5000 on your host. 

`docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"`


Use kubectl port-forward to map your local workstation to the minikube vm

`kubectl port-forward --namespace kube-system service/registry 5000:80`

Verify that you are able to access the minikube registry by running:

`curl http://localhost:5000/v2/_catalog`

### Backend

Backend app is located in backend folder. It is a simple app which accepts websocket connection and sends back simple JSON structure containing the pod name, namespace name and counter. `build.sh` script builds the app and pushes it to the registry. So, make backend the current directory and run

`build.sh`

`kubectl apply -f backend.yaml`

`kubectl get pods`
 
It gives you something like this
```
NAME                     READY   STATUS    RESTARTS        AGE
backend-0                1/1     Running   0               3d14h
backend-1                1/1     Running   0               3d14h
backend-2                1/1     Running   0               3d14h
backend-3                1/1     Running   0               3d14h
```
Create headless service. It will register backend pods in DNS
`kubectl apply -f headless.yaml`

Next step is optional. You can create load balancer and access backend from your host.
`kubectl apply -f svc.yaml`
`kubectl get svc` will show this line

```
NAME          TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
...
cloud-lb      LoadBalancer   10.103.79.71   <EXT-IP>     9000:30825/TCP               3d
...
```
Now you have to run `minikube tunnel` in separate console. It should ask you for password eventually.
Any command line websocket client will give you:
```
> websocat ws://localhost:9000/ws/ns-01
{"server":"backend-0","counter":1,"ns":"ns-01"}
{"server":"backend-0","counter":2,"ns":"ns-01"}
...
```

You bypass the router and connect to the backend directly.




run API proxy
k proxy --port=8080

access backend from the host (for debugging)



### The router

Now let's run the router. The router is located in 'router' folder. There is build.sh file which builds image and pushes it to the local registry. You have to run  `build.sh`

The router pod has two containers - nginx which terminates SSL and forwards packets to the router itself. 
The router app (main.go) accepts incoming connections, extracts parameter from the path (it expects /ws/<namespace>. In real life it should get hostname from the header and then extract namespace from this hostname). Also this app watches the configmap which has the routing table. This routing table overwrites routing rules calculated by applying simple hash algorithm.

So, you create overwriting configmap:
`kubectl -f overwriting.yaml`

Then you have to create configmap with certificates needed by nginx. Run 
`kubectl create configmap nginx-tls --from-file=../cert/cert`
This is not a good practice. You are not supposed to store any sensitive data in configmap. But for demo it is fine.

The router need access to k8s API (to watch configmap). To make it possible we have to create a cluster role

`kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default`

The app config file is `router.yaml`. This is a deployment configuration. Next step is to run
`kubectl -f router.yaml`

If everything is all right you will see this output for `kubctl get pods`

```
NAME                     READY   STATUS    RESTARTS   AGE
backend-0                1/1     Running   0          3d15h
backend-1                1/1     Running   0          3d15h
backend-2                1/1     Running   0          3d15h
backend-3                1/1     Running   0          3d15h
nginx-54485899d7-x6wvd   2/2     Running   0          138m
```
And finally create the LoadBalancer - `kubectl apply -f lb.yaml`

### The client

The test client is a simple React app located in test-client folder. Make this folder the current and run
`npm install`

Before running the client you need two things:
1. add root certificate to Crome (or any other browser) and make it trustable. The location of this self signed certificated is k8s-play/cert/rootCA.pem

2. The host certificate used by nginx is created for site www.mydomain.com. So, you have to create an entry in your hosts file for this name.

First at all you need to get ip address of the router load balancer. Run

`kubectl get svc`
It will show you 
```
NAME          TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
...
nginx         LoadBalancer   10.102.61.17   <LBIP>     443:32587/TCP,80:30307/TCP   3d1h
...
```
put <LBIP> into you /etc/hosts file:

```
...
<LBIP>  www.mydomain.com mydomain.com
...
```
Now you can run `npm start`

The client app has one table. Every row represents one connection. It has one cell which is not empty. The corresponding column name is the backend server accepted the connection for namespace (second column).
The value of this cell is router pod id which passed through this connection.

### The router in action

The namespace distribution over backend servers is random. You can edit ./router/overwriting.yaml file ang create a rule:

```
...
data:
  ns-0: "backend-2"
  ns-1: "backend-2"
  ns-2: "backend-2"
```

The file above will forward ns-0, ns-1, ns-2 namespaces to the same server (backend-2). 
Run `kubectl apply -f overwriting.yaml` and watch how green cells will jump to backend-2 column.





