## Setup local registry

We need a registry to push images to k8s. minikube allows to use insecure registry:

`minikube start --insecure-registry "10.0.0.0/24"`

Enable the registry addon to allow Docker to push images to minikube's registry:

`minikube addons enable registry`

In a separate terminal, redirect port 5000 from Docker to port 5000 on your host. 
`docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"`


Verify that you are able to access the minikube registry by running:

`curl http://localhost:5000/v2/_catalog`



kubectl port-forward --namespace kube-system service/registry 5000:80

run API proxy
k proxy --port=8080

accsess backend from the host (for debuging)
k apply -f svc.yaml
minikube tunnel

## The router
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default

create configmap for nginx. 



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


