package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	response_buffer := make([]byte, 30000)
	content_length, err := conn.Read(response_buffer)
	if err != nil {
		fmt.Println("Error reading response: ", err.Error())
	}

	response_parts := bytes.Split(response_buffer[:content_length], []byte("\r\n"))
	request_line := response_parts[0]

	request_line_parts := bytes.Split(request_line, []byte(" "))
	// http_method := request_line_parts[0]
	target := request_line_parts[1]
	// http_version := request_line_parts[2]

	if string(target) == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
