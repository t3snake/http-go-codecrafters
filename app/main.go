package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
)

const content_type_formatter = "Content-Type: %s\r\n"
const content_type_plaintext = "text/plain"

const content_length_formatter = "Content-Length: %d\r\n"

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

	request_buffer := make([]byte, 30000)
	content_length, err := conn.Read(request_buffer)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
	}

	request_parts := bytes.Split(request_buffer[:content_length], []byte("\r\n"))
	request_line := request_parts[0]

	request_line_parts := bytes.Split(request_line, []byte(" "))
	// http_method := request_line_parts[0]
	target := string(request_line_parts[1])
	// http_version := request_line_parts[2]

	if target == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(target, "/echo/") {
		response_content := strings.ReplaceAll(target, "/echo/", "")

		content_type := fmt.Sprintf(content_type_formatter, content_type_plaintext)
		content_length := fmt.Sprintf(content_length_formatter, len(response_content))

		response_headers := fmt.Sprintf("%s%s", content_type, content_length)

		response := fmt.Sprintf("HTTP/1.1 200 OK\r\n%s\r\n%s", response_headers, response_content)
		conn.Write([]byte(response))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
