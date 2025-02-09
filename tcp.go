package main

import (
	"errors"
	"io"
	"log"
	"net"

	"github.com/spf13/cobra"
)

func tcpMode(cmd *cobra.Command, args []string) {
	localAddr, err := cmd.Flags().GetString("local-address")
	if err != nil {
		log.Fatalf("error parsing flags: %s\n", err)
	}
	remoteAddr, err := cmd.Flags().GetString("remote-address")
	if err != nil {
		log.Fatalf("error parsing flags: %s\n", err)
	}
	if _, err := net.ResolveTCPAddr("tcp", localAddr); err != nil {
		log.Fatalf("invalid address: %s: %v\n", localAddr, err)
	}
	if _, err := net.ResolveTCPAddr("tcp", remoteAddr); err != nil {
		log.Fatalf("invalid address: %s: %v\n", remoteAddr, err)
	}

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
	go log.Printf("handling connection for %s\n", src.RemoteAddr())
	dst, err := net.Dial("tcp", destAddr)
	if err != nil {
		log.Printf("Failed to connect from %s to remote: %s: %s\n",
			src.RemoteAddr().String(),
			destAddr,
			err)
		return
	}
	defer dst.Close()

	done1 := make(chan struct{})
	done2 := make(chan struct{})

	go transfer(src, dst, done1)
	go transfer(dst, src, done2)

	select {
	case _ = <-done1:
	case _ = <-done2:
	}
}

func transfer(src net.Conn, dst net.Conn, done chan struct{}) {
	defer close(done)
	n, err := io.Copy(dst, src)
	if errors.Is(err, net.ErrClosed) {
		log.Printf("connection closed: %s -> %s. bytes written: %d\n",
			src.RemoteAddr().String(),
			dst.RemoteAddr().String(),
			n)
		return
	} else if err != nil {
		log.Printf("error copying from %s to %s: %s\n",
			src.RemoteAddr().String(),
			dst.RemoteAddr().String(),
			err)
		return
	}

	log.Printf("wrote %d bytes from %s to %s\n",
		n,
		src.RemoteAddr().String(),
		dst.RemoteAddr().String())
}
