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
import "math/rand"
import "bufio"

func reconciliateNodes(newNodeMap map[string]int) {
	for k, v := range newNodeMap {
		if nodeMap[k] > 0 {
			nodeMap[k] = (v + nodeMap[k]) / 2
		} else {
			nodeMap[k] = v
			nodeMapKeys = append(nodeMapKeys, k)
		}
	}
}

func handleConnection(conn net.Conn) {
	var out []byte
	out = make([]byte, 4)
	_, err := io.ReadFull(conn, out)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		log.Fatal("Shit", err)
	}

	log.Print("Got: ", string(out[:4]))

	lineScanner := bufio.NewScanner(conn)

	switch string(out[:4]) {
	case "JOIN":
		log.Print("got a join")
		for lineScanner.Scan() {
			addrStr := strings.TrimSpace(string(lineScanner.Bytes()))
			log.Print("server is joining: " + addrStr)
			nodeMap[addrStr] = initialNodeHealth
			nodeMapKeys = append(nodeMapKeys, addrStr)
		}
	case "PING":
		log.Print("got a ping")
		recievedNodes := make(map[string]int)
		for lineScanner.Scan() {
			server := string(lineScanner.Bytes())
			addressAndHealth := strings.Split(server, "-")
			if len(addressAndHealth) != 2 {
				break
			}
			health, e := strconv.ParseInt(strings.TrimSpace(addressAndHealth[1]), 10, 64)
			if e != nil {
				log.Print("corrupt packet ", e)
				break
			}
			recievedNodes[addressAndHealth[0]] = int(health)
		}
		reconciliateNodes(recievedNodes)
		log.Print("deserialized: ", recievedNodes)
		conn.Write([]byte(serializeNodes(nodeMap)))
	default:
		log.Print("Unsupported action")
	}
	conn.Close()
}

// TODO: implement non-string based protocol parsing
func serializeNodes(nodeMap map[string]int) string {
	nodes := ""
	log.Print("Serializing: ", nodeMap)
	for k, v := range nodeMap {
		nodes = nodes + k + "-" + strconv.Itoa(v) + "\n"
	}
	return nodes
}

// TODO: implement non-string based protocol parsing
func deSerializeNodes(nodesString string) map[string]int {
	deserializedNodeMap := make(map[string]int)
	servers := strings.Split(nodesString, ";")
	for _, s := range servers {
		addressAndHealth := strings.Split(s, "-")
		if len(addressAndHealth) != 2 {
			return deserializedNodeMap
		}
		health, e := strconv.ParseInt(strings.TrimSpace(addressAndHealth[1]), 10, 64)
		if e != nil {
			log.Print("corrupt packet", e)
			return deserializedNodeMap
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
	nodes := serializeNodes(nodeMap)
	log.Print("pinging", nodes)
	con.Write([]byte("PING" + nodes))
	//handleConnection(con)
	con.Close()
}

func join(masterAddr string, myAddr string) {
	con, err := net.Dial("tcp", masterAddr)
	if err != nil {
		log.Print("Couldn't dial", err)
	} else {
		log.Print("joining")
		con.Write([]byte("JOIN" + myAddr))
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

func pingOneInMap() {
	index := 0
	if len(nodeMapKeys) > 1 {
		index = rand.Intn(len(nodeMapKeys))
	}
	address := nodeMapKeys[index]
	log.Print("Lets ping " + address)
	con, err := net.Dial("tcp", address)
	if err != nil {
		log.Print("Couldn't dial", err)
		nodeMap[address] = nodeMap[address] - 1
	} else {
		go ping(con)
	}
}

func setupPinger(nodeMap map[string]int) {
	for {
		time.Sleep(5000 * time.Millisecond)
		//pingOverList(nodeList.Front())
		pingOneInMap()
	}
}

// store nodes in a map
// TODO: add some sort of mutex to lock for concurrent access
var nodeMap map[string]int

// to pick a random node
var nodeMapKeys []string

const initialNodeHealth = 10

var nodeList list.List

var nodes [100]Server
var lastnode int

func main() {

	var peerIp = flag.String("peerIp", "0.0.0.0", "IP of server")
	var peerPort = flag.Int("peerPort", 6969, "Port of the initial peer to connect to")
	var port = flag.Int("port", 6969, "Port to listen on")
	var ip = flag.String("ip", "0.0.0.0", "IP of server")
	flag.Parse()

	portString := strconv.Itoa(*port)
	peerPortString := strconv.Itoa(*peerPort)

	log.Print("Master port is" + peerPortString)
	log.Print("Starting server on: " + portString)

	nodeMap = make(map[string]int)
	nodeMap[*peerIp+":"+strconv.Itoa(*peerPort)] = initialNodeHealth
	nodeMapKeys = make([]string, 0)
	nodeMapKeys = append(nodeMapKeys, *ip+":"+portString)

	ln, err := net.Listen("tcp", ":"+os.Args[2])
	if err != nil {
		log.Fatal("Shit", err)
	}
	go join(*peerIp+":"+peerPortString, *ip+":"+portString)
	go setupPinger(nodeMap)
	for {
		listen(ln)
	}

}
