package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	isServerPtr := flag.Bool("server", false, "server mode")
	address := flag.String("addr", "", "address for server listen or client connect")
	flag.Usage = usage
	flag.Parse()

	if address == nil || len(*address) == 0 {
		log.Fatal("Please provide address, format like: [host]:port.")
	}

	if *isServerPtr {
		err := server(*address)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		err := client(*address)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options], options list:\n", os.Args[0])
	flag.PrintDefaults()
}

func client(address string) error {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		if _, err := fmt.Fprintf(c, text+"\n"); err != nil {
			return err
		}

		message, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			return err
		}
		log.Print("<< " + message)
		if strings.TrimSpace(string(text)) == "STOP" {
			log.Println("TCP client exiting...")
			break
		}
	}

	return nil
}

func server(address string) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("listening on %s", l.Addr())

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		log.Println("accepted incoming connection from: ", c.RemoteAddr())
		go serverAcceptHandler(c)
	}
}

func serverAcceptHandler(c net.Conn) error {
	defer func() {
		log.Println("serverAcceptHandler defer")
		c.Close()
	}()

	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(netData)) == "STOP" {
			log.Println("Exiting TCP server!")
			c.Write([]byte("Good bye.\n"))
		}

		log.Print("<< ", string(netData))
		t := time.Now()
		myTime := t.Format(time.RFC3339) + "\n"
		c.Write([]byte(myTime))
	}
}
