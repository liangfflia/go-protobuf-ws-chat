package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/oyvindsk/go-protobuf-ws-chat/lib/message"
)

func main() {

	//fmt.Println(message.MessageType_JOINROOM)

	r := mux.NewRouter()
	r.HandleFunc("/ws", handleWebsocket)

	fmt.Println("Listening..")
	err := http.ListenAndServe(":8080", r)

	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}

}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {

	// Deny all but HTTP GET
	if r.Method != "GET" {
		// FIXME log.WithField("method", r.Method).Error("Disallowed http method")
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Upgrade connection to Websocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // FIXME : Remove
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// FIXME log.Error("Upgrading to websockets failed:", err)
		http.Error(w, "Error Upgrading to websockets", 400)
		return
	}

	msgtype, err := doHandshake(ws)
	if err != nil {
		log.Println("Handshake failed:", err)
		ws.WriteMessage(msgtype, []byte(err.Error()))
		log.Println("Killing connection to client: Error above")
		return
	}

	// Nice! Switch to proto3
	msg := message.Message{}
	msgtype, data, err := ws.ReadMessage()
	err = proto.Unmarshal(data, &msg)
	if err != nil {
		log.Fatal(err)
	}

	// we expect the first msg to be Register
	switch msg.Type {
	case message.MessageType_REGISTERNICK:
		reg := message.RegisterNick{}
		err = proto.Unmarshal(msg.Content, &reg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Registering: %s", reg.Nick)
	default:
		log.Fatal("Register expected. got:", msg.Type)

	}

	log.Printf("GOT: %+v\n%T\n", msg, msg)
	log.Printf("Err: %s\n%#v\n", err, err)

	log.Println("Killing connection to client: EOF")
}

func doHandshake(ws *websocket.Conn) (int, error) {
	// Expect this to go like this:
	// >> YO! 1
	// << OY! 1
	// >> OK

	// TODO: Timeout! Somewhere, here rr at another level?

	msgtype, msg, err := ws.ReadMessage()
	log.Printf(">>> %s", msg)
	if err != nil || !bytes.HasPrefix(msg, []byte("YO! ")) {
		return msgtype, fmt.Errorf("handshake error: Expecting 'YO! [version]'. Got: %s. Err: %s", msg, err)
	}

	// Ignore versioning. Since we only support version 1 we always send back the same answer =)
	ws.WriteMessage(msgtype, []byte("OY! 1"))
	log.Printf("<<< %s", "OY! 1")

	msgtype, msg, err = ws.ReadMessage()
	log.Printf(">>> %s", msg)
	if err != nil || !bytes.Equal(msg, []byte("OK")) {
		return msgtype, fmt.Errorf("handshake error: Expecting 'OK', got: %s. Err: %s", msg, err)
	}

	return 0, nil
}
