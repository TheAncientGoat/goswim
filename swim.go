package main

import "net"
import "io"
import "os"
import "log"
import "time"
import "strconv"

func handleConnection(conn net.Conn) {
	var out []byte
	out = make([]byte, 100)
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
		log.Print("port is joining" + portStr)
		nodes[lastnode] = Server{"0.0.0.0", int(p), 1000}
		lastnode += 1
		for _, i := range nodes {
			log.Print(i.port)
		}
	}
	conn.Close()
}

func ping(con net.Conn) {
	log.Print("pinging")
	con.Write([]byte("PING"))
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

func setupPinger(port string) {
	for {
		time.Sleep(10000 * time.Millisecond)
		for _, node := range nodes {
			if node.port > 0 {
				con, err := net.Dial("tcp", "localhost:"+strconv.Itoa(node.port))
				if err != nil {
					log.Print("Couldn't dial", err)
				} else {
					go ping(con)
				}
			}
		}
	}
}

var nodes [100]Server
var lastnode int

func main() {
	port := os.Args[1]
	peerPort, e := strconv.ParseInt(os.Args[2], 10, 64)
	if e != nil {
		// handle
	}
	nodes[0] = Server{"0.0.0.0", int(peerPort), 1000}
	lastnode = 0

	ln, err := net.Listen("tcp", ":"+os.Args[2])
	if err != nil {
		log.Fatal("Shit", err)
	}
	go join(port, os.Args[2])
	go setupPinger(port)
	for {
		listen(ln)
	}

}
