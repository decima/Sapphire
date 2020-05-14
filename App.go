package main

import (
	"fmt"
	"github.com/docker/docker/api/types/events"
	"github.com/gorilla/websocket"
	"sapphire/docker"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var Connections = map[int64]*websocket.Conn{}
var DockerData docker.Response

type WSMessage struct {
	Action  string
	Content interface{}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)
	go LoadRefresh()
	go CatchEvents()
	log.Println("starting server on :8000")
	panic(http.ListenAndServe(":8000", nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadFile("index.html")
	if err != nil {
		fmt.Println("Could not open file.", err)
	}
	fmt.Fprintf(w, "%s", content)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Could not open websocketHandler connection", http.StatusBadRequest)
	}
	unixNano := time.Now().UnixNano()
	Connections[unixNano] = conn
	go wsApp(unixNano, conn)
}
func LoadRefresh() {
	for {
		DockerData = docker.GetData()
		time.Sleep(5 * time.Second)
	}
}
func wsApp(index int64, conn *websocket.Conn) {
	sendInfo(conn, WSMessage{
		Action:  "init",
		Content: DockerData,
	})

}

func sendInfo(conn *websocket.Conn, message interface{}) bool {

	if err := conn.WriteJSON(message); err != nil {
		return false
	}
	return true
}

func SendToAll(message interface{}) {
	for clientId, conn := range Connections {
		if !sendInfo(conn, message) {
			delete(Connections, clientId)

		}
	}

}
func CatchEvents() {
	msg, _ := docker.CatchEvents()
	for message := range msg {

		switch message.Type {
		case "service":
			HandleService(message)
			break
		case "network":
			HandleNetwork(message)
			break
		case "container":
			HandleContainer(message)
			break
		}
	}

}

func HandleNetwork(message events.Message) {
	switch message.Action {
	case "create":
		SendToAll(WSMessage{
			Action:  "network.create",
			Content: message.Actor.Attributes["name"],
		})
		break
	case "connect":
		containerId := message.Actor.Attributes["container"]
		serviceName := docker.GetContainerServiceName(containerId)
		if serviceName != nil {
			SendToAll(WSMessage{
				Action: "network.connect",
				Content: struct {
					Network string
					Service string
				}{message.Actor.Attributes["name"], *serviceName},
			})
		}
		break
	case "destroy":
		SendToAll(WSMessage{
			Action:  "network.destroy",
			Content: message.Actor.Attributes["name"],
		})
		break
	}
}

func HandleService(message events.Message) {
	switch message.Action {
	case "create":
		SendToAll(WSMessage{
			Action:  "service.create",
			Content: message.Actor.Attributes["name"],
		})
		break
	case "remove":
		SendToAll(WSMessage{
			Action:  "service.remove",
			Content: message.Actor.Attributes["name"],
		})
		break
	}
}

func HandleContainer(message events.Message) {
	if message.Action == "start" || message.Action == "stop" {
		service := message.Actor.Attributes["com.docker.swarm.service.name"]
		node := message.Actor.Attributes["com.docker.swarm.servicenode.id"]
		name := message.Actor.Attributes["name"]

		SendToAll(WSMessage{
			Action: "container." + message.Action,
			Content: struct {
				Service   *string
				Container *string
				Node      *string
			}{&service, &name, &node},
		})
	}
}
