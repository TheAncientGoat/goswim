package main

import "net"
import "io"
import "os"
import "log"
import "time"
import "strconv"
import "flag"
import "container/list"
import "strings"

func reconciliateNodes(newNodeMap map[string]int) {
	for k, v := range newNodeMap {
		if nodeMap[k] > 0 {
			nodeMap[k] = (v + nodeMap[k]) / 2
		} else {
			nodeMap[k] = v
		}
	}
}

func handleConnection(conn net.Conn) {
	var out []byte
	out = make([]byte, 1000)
	n, err := io.ReadFull(conn, out)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		log.Fatal("Shit", err)
	}

	log.Print(string(out[:100]), n)

	switch string(out[:4]) {
	case "JOIN":
		log.Print("Found a join")
		portStr := string(out[5:9])
		p, e := strconv.ParseInt(portStr, 10, 0)
		if e != nil {
			log.Print("Cannot parse", e)
		}
		log.Print("port is joining: " + portStr)
		nodeList.PushBack(Server{"0.0.0.0", int(p), 1000})
		nodeMap["0.0.0.0:"+portStr] = 1000
	case "PING":
		log.Print("got a ping")
		recievedNodes := deSerializeNodes(string(out[4:len(out)]))
		log.Print(recievedNodes)
		reconciliateNodes(recievedNodes)
		conn.Write([]byte(serializeNodes(nodeMap)))
	default:
		log.Print("Unsupported action")
	}
	conn.Close()
}

func serializeNodes(nodeMap map[string]int) string {
	nodes := ""
	for k, v := range nodeMap {
		nodes = nodes + k + "-" + strconv.Itoa(v) + ";"
	}
	return nodes
}

func deSerializeNodes(nodesString string) map[string]int {
	deserializedNodeMap := make(map[string]int)
	servers := strings.Split(nodesString, ";")
	for _, s := range servers {
		addressAndHealth := strings.Split(s, "-")
		if len(addressAndHealth) != 2 {
			return deserializedNodeMap
		}
		health, e := strconv.ParseInt(addressAndHealth[1], 10, 64)
		if e != nil {
			log.Print("corrupt packet", e)
			return nil
		}
		deserializedNodeMap[addressAndHealth[0]] = int(health)
	}
	return deserializedNodeMap
}

func serializeNodesList(elem *list.Element, hostsString string) string {
	server := elem.Value.(Server)
	hostsString = hostsString + ";" + server.ip + "-" + strconv.Itoa(server.port)
	if elem.Next() != nil {
		return serializeNodesList(elem.Next(), hostsString)
	}
	return hostsString
}

func ping(con net.Conn) {
	log.Print("pinging")
	con.Write([]byte("PING" + serializeNodes(nodeMap)))
	//handleConnection(con)
	con.Close()
}

func join(masterPort string, myPort string) {
	con, err := net.Dial("tcp", "localhost:"+masterPort)
	if err != nil {
		log.Print("Couldn't dial", err)
	} else {
		log.Print("joining")
		con.Write([]byte("JOIN:" + myPort))
		con.Close()
	}
}

type Server struct {
	ip     string
	port   int
	health int
}

func listen(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		log.Fatal("Shit", err)
	}
	go handleConnection(conn)
}

func pingOverList(elem *list.Element) {
	var node Server
	node = elem.Value.(Server)
	if node.health > 0 {
		con, err := net.Dial("tcp", "localhost:"+strconv.Itoa(node.port))
		if err != nil {
			log.Print("Couldn't dial", err, node.health)
			node.health -= 1
		} else {
			go ping(con)
		}
	}
	if elem.Next() != nil {
		pingOverList(elem.Next())
	}
}

func pingOverMap(nodeMap map[string]int) {
	for address, health := range nodeMap {
		con, err := net.Dial("tcp", address)
		if err != nil {
			log.Print("Couldn't dial", err, health)
			nodeMap[address] = health - 1
		} else {
			go ping(con)
		}
	}
}

func setupPinger(nodeMap map[string]int) {
	for {
		time.Sleep(5000 * time.Millisecond)
		//pingOverList(nodeList.Front())
		pingOverMap(nodeMap)
	}
}

var nodeMap map[string]int
var nodeList list.List

var nodes [100]Server
var lastnode int

func main() {
	var peerPort = flag.Int("peerPort", 6969, "Port of the initial peer to connect to")
	var port = flag.Int("port", 6969, "Port to listen on")
	flag.Parse()

	nodeList.Init()
	nodeList.PushBack(Server{"0.0.0.0", *peerPort, 1000})

	portString := strconv.Itoa(*port)
	peerPortString := strconv.Itoa(*peerPort)

	log.Print("Master port is" + peerPortString)
	log.Print("Starting server on: " + portString)

	nodeMap = make(map[string]int)
	nodeMap["0.0.0.0:"+strconv.Itoa(*peerPort)] = 1000

	ln, err := net.Listen("tcp", ":"+os.Args[2])
	if err != nil {
		log.Fatal("Shit", err)
	}
	go join(peerPortString, portString)
	go setupPinger(nodeMap)
	for {
		log.Print("listening again")
		listen(ln)
	}

}
