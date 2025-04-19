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
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go HandleConnection(conn)
	}
}

func HandleConnection(conn net.Conn) {
	request_buffer := make([]byte, 30000)
	request_length, err := conn.Read(request_buffer)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
	}

	request_line, request_body, request_headers := ParseRequest(request_buffer[:request_length])

	fmt.Printf("Request Body: %s", request_body)

	http_method, url, _ := ParseRequestLine(request_line)

	if http_method == "GET" {
		if url == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

		} else if strings.HasPrefix(url, "/echo/") {
			response_content := strings.ReplaceAll(url, "/echo/", "")

			response := GenerateResponse(response_content, content_type_plaintext, "HTTP/1.1 200 OK")
			conn.Write([]byte(response))

		} else if url == "/user-agent" {
			response_content := request_headers["User-Agent"]

			response := GenerateResponse(response_content, content_type_plaintext, "HTTP/1.1 200 OK")
			conn.Write([]byte(response))
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))

		}
	}
}

// ParseRequest takes the request string and returns request line as string, request body as string and map of request headers respectively
func ParseRequest(request []byte) (string, string, map[string]string) {
	request_parts := bytes.Split(request, []byte("\r\n"))
	request_line := string(request_parts[0])
	request_body := string(request_parts[len(request_parts)-1])

	request_headers := make(map[string]string)

	// last header ends with \r\n and immediately \r\n for separation between header si ignore the second last split
	for i := 1; i < len(request_parts)-2; i++ {
		header := string(request_parts[i])
		key_val := strings.Split(header, ": ")

		request_headers[key_val[0]] = key_val[1]
	}

	return request_line, request_body, request_headers

}

// ParseRequestLine takes the request line and returns the http verb, URL, http version respectively
func ParseRequestLine(request_line string) (string, string, string) {
	request_line_parts := strings.Split(request_line, " ")
	http_method := request_line_parts[0]
	target := string(request_line_parts[1])
	http_version := request_line_parts[2]

	return http_method, target, http_version
}

// GenerateResponse takes the content, the content type and response to generate and return response
func GenerateResponse(content, content_type, response_line string) string {
	content_type_header := fmt.Sprintf(content_type_formatter, content_type)
	content_length_header := fmt.Sprintf(content_length_formatter, len(content))

	response_headers := fmt.Sprintf("%s%s", content_type_header, content_length_header)

	response := fmt.Sprintf("%s\r\n%s\r\n%s", response_line, response_headers, content)
	return response
}
