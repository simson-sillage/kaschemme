package main

import (
	"io"
	"log"
	"net"
)

func main() {
	localAddr := ":8080"
	remoteAddr := "localhost:8081"
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("failed to bind to %s: %s\n", localAddr, err)
	}
	defer listener.Close()
	log.Printf("Listening on: %s\n", localAddr)
	log.Printf("Forwarding to: %s\n", remoteAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection error: %s\n", err)
			continue
		}
		go handleConnection(conn, remoteAddr)
	}
}

func handleConnection(src net.Conn, destAddr string) {
	defer src.Close()
	log.Printf("handling connection for %s\n", src.RemoteAddr())
	dst, err := net.Dial("tcp", destAddr)
	if err != nil {
		log.Printf("Failed to connect to remote: %s: %s\n", destAddr, err)
		return
	}
	defer dst.Close()
	go transfer(src, dst)
	transfer(dst, src)

}

func transfer(src net.Conn, dst net.Conn) {
	n, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("error copying from %s to %s: %s\n",
			src.RemoteAddr().String(),
			dst.RemoteAddr().String(),
			err)
		return
	}
	go log.Printf("wrote %d bytes from %s to %s\n",
		n,
		src.RemoteAddr().String(),
		dst.RemoteAddr().String())
}
