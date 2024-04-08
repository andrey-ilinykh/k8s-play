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

> websocat ws://localhost:9000/ws/ns-01
{"server":"backend-0","counter":1,"ns":"ns-01"}
{"server":"backend-0","counter":2,"ns":"ns-01"}
...
You bypass the router and connect to the backend directly.

### The router

Now let's run the router. 

run API proxy
k proxy --port=8080

access backend from the host (for debugging)



## The router
create a role 
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default

create configmap for nginx. Assuming you are in the parent folder run.

k create configmap nginx-tls --from-file=cert/cert

cert/cert folder contains the certificate and the private key. This is not good practice to use configmap for sensitive information. It is acceptable for prototype only.

in router folder
k apply -f overwriting.yaml

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
  -nodes -keyout mydomain.com.key -out mydomain.com.crt -extensions san -config \
  <(echo "[req]"; 
    echo distinguished_name=req; 
    echo "[san]"; 
    echo subjectAltName=DNS:mydomain.com,DNS:*.mydomain.com
    ) \
  -subj "/CN=mydomain.com"


Accept-Encoding:
gzip, deflate, br, zstd
Accept-Language:
en-US,en;q=0.9,ru;q=0.8
Cache-Control:
no-cache
Connection:
Upgrade
Host:
www.mydomain.com
Origin:
http://localhost:3000
Pragma:
no-cache
Sec-Websocket-Extensions:
permessage-deflate; client_max_window_bits
Sec-Websocket-Key:
Px99yEVWj3yYVW3P8QsBnw==
Sec-Websocket-Version:
13
Upgrade:
websocket
User-Agent:
Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36


