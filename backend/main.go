package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	//addr      = flag.String("addr", ":8080", "http service address")
	//homeTempl = template.Must(template.New("").Parse(homeHTML))
	//filename string
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Message struct {
	Server    string `json:"server"`
	Counter   int    `json:"counter"`
	Namespace string `json:"ns"`
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	io.WriteString(w, fmt.Sprintf("This is my website! I'm %s\n", os.Getenv("HOSTNAME")))
}
func getHello(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /hello request\n")
	io.WriteString(w, "Hello, HTTP!\n")
}
func reader(ws *websocket.Conn) {
	defer ws.Close()
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reading error %v\n", err)
			break
		}
	}
}
func writer(ws *websocket.Conn, ns string, header map[string][]string) {
	defer ws.Close()
	n := 0
	for {
		n++

		m := Message{Server: hostName, Counter: n, Namespace: ns}
		str, err := json.Marshal(m)
		//str, err := json.Marshal(header)
		if err != nil {
			continue
		}
		err = ws.WriteMessage(websocket.TextMessage, []byte(str))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Writing error %v\n", err)
			break
		}
		time.Sleep(1 * time.Second)
	}
}
func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			fmt.Fprintf(os.Stderr, "Can't start server %v\n", err)
		}
		return
	}
	ns := mux.Vars(r)["ns"]
	fmt.Printf("Headers:\n %v\n", r)
	go writer(ws, ns, r.Header)
	reader(ws)
}

var hostName string

func main() {
	hostName = os.Getenv("HOSTNAME")
	r := mux.NewRouter()
	r.HandleFunc("/", getRoot)
	r.HandleFunc("/hello", getHello)
	r.HandleFunc("/ws/{ns}", serveWs)

	err := http.ListenAndServe(":3333", r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't start server %v\n", err)
		os.Exit(-1)
	}

}
