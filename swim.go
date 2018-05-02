package main

import "net"
import "io"
import "os"
import "log"
import "time"

func handleConnection(conn net.Conn) {
	var out []byte
	out = make([]byte, 4)
	n, err := io.ReadFull(conn, out)
	if err != nil && err != io.EOF {
		log.Fatal("Shit", err)
	}

	log.Print(string(out[:4]), n)
	conn.Close()
}

func ping(con net.Conn) {
	log.Print("pinging")
	con.Write([]byte("PING"))
	con.Close()
}

type Server struct {
	ip     net.IPAddr
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
		time.Sleep(1000 * time.Millisecond)
		con, err := net.Dial("tcp", "localhost:"+port)
		if err != nil {
			log.Print("Couldn't dial", err)
		} else {
			go ping(con)
		}
	}
}

func main() {
	port := os.Args[1]
	var nodes [1]Server
	addr := net.IPAddr{IP: net.IP("0.0.0.0:" + port)}
	nodes[0] = Server{addr, 123, 1000}

	ln, err := net.Listen("tcp", ":"+os.Args[2])
	if err != nil {
		log.Fatal("Shit", err)
	}
	go setupPinger(port)
	for {
		listen(ln)
	}

}
