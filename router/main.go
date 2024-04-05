package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//klog "k8s.io/klog/v2"
)

type Proxy struct {
	ns      string
	dest    string
	inConn  *websocket.Conn
	outConn *websocket.Conn
}

type Destination struct {
	name      string
	proxyList []*Proxy
}

func (d *Destination) addProxy(p *Proxy) {
	d.proxyList = append(d.proxyList, p)
}

func (d *Destination) close() {
	fmt.Printf("close: %v\n", d)
	for _, p := range d.proxyList {
		p.Close()
	}
}

func (d *Destination) removeProxy(p *Proxy) {
	nslise := make([]*Proxy, 0, len(d.proxyList))
	for _, e := range d.proxyList {
		if e != p {
			nslise = append(nslise, e)
		}
	}
	d.proxyList = nslise
}

func (d *Destination) isEmpty() bool {
	return len(d.proxyList) == 0
}

func (p *Proxy) Close() {
	p.inConn.Close()
	p.outConn.Close()
}

var (
	//addr      = flag.String("addr", ":8080", "http service address")
	//homeTempl = template.Must(template.New("").Parse(homeHTML))
	//filename string
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	dialer    = websocket.DefaultDialer
	dstPort   string
	dstDomain string
	dstPrefix string
	hostName  string
)

// func manager(dstChan chan chan string, npChan chan *Proxy, errChan chan *Proxy, mapChan <-chan map[string]string) {
func manager(mapChan <-chan map[string]string) (chan chan string, chan *Proxy, chan *Proxy) {
	dstChan := make(chan chan string)
	npChan := make(chan *Proxy)
	errChan := make(chan *Proxy)

	proxyMap := make(map[string]Destination)
	oMap := make(map[string]string)
	destName := func(ns string) string {
		if dst, ok := oMap[ns]; ok {
			return fmt.Sprintf("%s%s", dst, dstDomain)

		}
		h := fnv.New32()
		h.Write([]byte(ns))
		//backend-0.backend-svc.default.svc.cluster.local
		return fmt.Sprintf("%s-%d%s", dstPrefix, h.Sum32()%4, dstDomain)
	}

	filterProxy := func() (map[string]Destination, []*Destination) {
		newMap := make(map[string]Destination)
		dsts := make([]*Destination, 0, 1024)
		for k, dst := range proxyMap {
			if dst.name == destName(k) {
				newMap[k] = dst
			} else {
				dsts = append(dsts, &dst)
			}
		}

		return newMap, dsts
	}

	go func() {
		for {
			select {
			case dst := <-dstChan:
				ns := <-dst
				dst <- destName(ns)

			case m := <-mapChan:
				oMap = m
				newMap, dsts := filterProxy()
				fmt.Printf("mapChan %v, %v\n", newMap, dsts)
				proxyMap = newMap
				for _, dst := range dsts {
					dst.close()
				}
				fmt.Printf("Got new map: %v\n", oMap)

			case np := <-npChan:
				// there is chance that overwriting map got changed and the destination is not valid anymore
				if np.dest == destName(np.ns) {
					dst := proxyMap[np.ns]
					dst.name = np.dest
					dst.addProxy(np)
					proxyMap[np.ns] = dst
					fmt.Printf("New proxy created %v. %v\n", np, proxyMap)
				} else {
					np.Close()
				}

			case ep := <-errChan:
				dst, ok := proxyMap[ep.ns]
				if ok {
					fmt.Printf("slice length= %d\n", len(dst.proxyList))
					dst.removeProxy(ep)
					if dst.isEmpty() {
						delete(proxyMap, ep.ns)
					}
					fmt.Printf("Proxy %v removed. %v\n", ep, proxyMap)
					ep.Close()
				}

			}
		}
	}()
	return dstChan, npChan, errChan
}

func reader(proxy *Proxy, errChan chan *Proxy) {
	//defer from.Close()
	//defer to.Close()
	for {
		proxy.inConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		mt, bytes, err := proxy.inConn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reading error %v\n", err)
			errChan <- proxy
			break
		}
		if err = proxy.outConn.WriteMessage(mt, bytes); err != nil {
			fmt.Fprintf(os.Stderr, "Writing error %v\n", err)
			errChan <- proxy
			break
		}
	}
}

func writer(proxy *Proxy, errChan chan *Proxy) {
	//defer from.Close()
	//defer to.Close()
	for {
		proxy.outConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		mt, bytes, err := proxy.outConn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "writer Reading error %v\n", err)
			errChan <- proxy
			break
		}
		var msg map[string]interface{}
		err = json.Unmarshal(bytes, &msg)

		if err == nil {
			msg["router"] = hostName
			newBytes, err := json.Marshal(msg)
			if err == nil {
				bytes = newBytes
			}
		}
		if err = proxy.inConn.WriteMessage(mt, bytes); err != nil {
			fmt.Fprintf(os.Stderr, "writer Writing error %v\n", err)
			errChan <- proxy
			break
		}
	}
}

func serveWs(dstChan chan chan string, newChan chan *Proxy, errChan chan *Proxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				fmt.Fprintf(os.Stderr, "Can't start server %v\n", err)
			}
			return
		}

		ns, ok := mux.Vars(r)["ns"]
		if !ok {
			fmt.Fprintf(os.Stderr, "<ns> paraneter is requiered \n")
			ws.Close()
			return

		}

		ch := make(chan string)
		dstChan <- ch
		ch <- ns
		dst := <-ch

		bUrl := fmt.Sprintf("ws://%s:%s/ws/%s", dst, dstPort, ns)
		bws, _, err := dialer.Dial(bUrl, http.Header{})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't connect to backend server %s %v\n", bUrl, err)
			ws.Close()
			return
		}

		np := Proxy{ns: ns, dest: dst, inConn: ws, outConn: bws}
		newChan <- &np
		go writer(&np, errChan)

		reader(&np, errChan)

	}

}

func cmWatcher(ctx context.Context, clientCfg *rest.Config, configName, namespace string) (<-chan map[string]string, error) {
	out := make(chan map[string]string)
	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create our clientset: %v", err)
	}

	cmi := clientset.CoreV1().ConfigMaps(namespace)
	_, err = cmi.List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Can't list cm:%v\n", err)
	}
	var wi watch.Interface
	wi, err = cmi.Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{Name: "overwriting", Namespace: "default"}))
	if err != nil {
		return nil, fmt.Errorf("unable to create watcher: %v", err)
	}
	go func() {
		for e := range wi.ResultChan() {
			if cm, ok := e.Object.(*v1.ConfigMap); ok {
				out <- cm.Data
			}
		}
	}()

	return out, nil

}

func getConfig() (*rest.Config, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "localhost" {
		userDir, err := os.UserHomeDir()
		if err != nil {
			panic(err.Error())
		}
		kubeconfig := userDir + "/.kube/config"
		fmt.Printf("kubeconfig=%s\n", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
func main() {

	config, err := getConfig()

	if err != nil {
		panic(err.Error())
	}

	hostName = os.Getenv("HOSTNAME")

	lstPort := os.Getenv("LISTEN_PORT")
	if lstPort == "" {
		panic("LISTEN_PORT env is required")
	}
	_, err = strconv.Atoi(lstPort)
	if err != nil {
		panic("Can't parse LISTEN_PORT " + lstPort)
	}

	dstPort = os.Getenv("DST_PORT")
	if dstPort == "" {
		panic("DST_PORT env is required")
	}
	_, err = strconv.Atoi(dstPort)
	if err != nil {
		panic("Can't parse DST_PORT " + dstPort)
	}

	dstPrefix = os.Getenv("DST_PREFIX")
	if dstPrefix == "" {
		panic("DST_PREFIX env is required")
	}

	dstDomain = os.Getenv("DST_DOMAIN")

	globalCtx, _ := context.WithCancel(context.Background())
	//configWatcher(globalCtx, config)
	mapCh, err := cmWatcher(globalCtx, config, "overwriting", "default")
	if err != nil {
		panic(err.Error())
	}
	dstChan, newChan, errChan := manager(mapCh)

	router := mux.NewRouter()
	router.HandleFunc("/ws/{ns}", serveWs(dstChan, newChan, errChan))
	http.ListenAndServe(":"+lstPort, router)
}
