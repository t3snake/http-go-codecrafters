package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const content_type_formatter = "Content-Type: %s\r\n"
const content_type_plaintext = "text/plain"
const content_type_octet = "application/octet-stream"

const content_length_formatter = "Content-Length: %d\r\n"
const content_encoding_formatter = "Content-Encoding: %s\r\n"

var accepted_compression = []string{"gzip"}

const http_200 = "HTTP/1.1 200 OK"
const http_201 = "HTTP/1.1 201 Created"
const http_404 = "HTTP/1.1 404 Not Found"

const timeout_duration time.Duration = time.Duration(1.5 * float64(time.Second))

var directory_flag *string

func main() {
	fmt.Println("Logs from your program will appear here!")

	directory_flag = flag.String("directory", "", "")
	flag.Parse()

	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	for {
		fmt.Println("Conn started")
		// conn.SetReadDeadline(time.Now().Add(timeout_duration))
		go HandleConnection(conn)
	}
}

func HandleConnection(conn net.Conn) {
	request_buffer := make([]byte, 30000)
	request_length, err := conn.Read(request_buffer)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}
	// conn.SetReadDeadline(time.Now().Add(timeout_duration))

	request_line, request_body, request_headers := ParseRequest(request_buffer[:request_length])
	if request_headers["connection"] == "close" {
		defer conn.Close()
	}

	fmt.Printf("Request Body: %s", request_body)

	http_method, url, _ := ParseRequestLine(request_line)

	if http_method == "GET" {
		if url == "/" {
			conn.Write(fmt.Appendf(nil, "%s\r\n\r\n", http_200))

		} else if strings.HasPrefix(url, "/echo/") {
			response_content := strings.ReplaceAll(url, "/echo/", "")

			response := GenerateResponse([]byte(response_content), content_type_plaintext, http_200, request_headers)
			conn.Write(response)

		} else if url == "/user-agent" {
			response_content := request_headers["User-Agent"]

			response := GenerateResponse([]byte(response_content), content_type_plaintext, http_200, request_headers)
			conn.Write(response)
		} else if strings.HasPrefix(url, "/files/") {
			filename := strings.ReplaceAll(url, "/files/", "")
			file_path := *directory_flag

			content, err := os.ReadFile(fmt.Sprintf("%s/%s", file_path, filename))
			if err != nil {
				conn.Write(fmt.Appendf(nil, "%s\r\n\r\n", http_404))
				return
			}
			response := GenerateResponse(content, content_type_octet, http_200, request_headers)
			conn.Write(response)

		} else {
			conn.Write(fmt.Appendf(nil, "%s\r\n\r\n", http_404))

		}
	} else if http_method == "POST" {
		if strings.HasPrefix(url, "/files/") {
			filename := strings.ReplaceAll(url, "/files/", "")
			file_path := *directory_flag

			err = os.WriteFile(fmt.Sprintf("%s/%s", file_path, filename), request_body, 0644)
			if err != nil {
				fmt.Println("Error writing file: ", err.Error())
				return
			}

			conn.Write(fmt.Appendf(nil, "%s\r\n\r\n", http_201))
		}
	}
}

// ParseRequest takes the request string and returns request line as string, request body as byte array and map of request headers respectively
func ParseRequest(request []byte) (string, []byte, map[string]string) {
	request_parts := bytes.Split(request, []byte("\r\n"))
	request_line := string(request_parts[0])
	request_body := request_parts[len(request_parts)-1]

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

// GenerateResponse takes the content, the content type, response line and request headers to generate response
func GenerateResponse(content []byte, content_type, response_line string, request_headers map[string]string) []byte {
	content_type_header := fmt.Sprintf(content_type_formatter, content_type)

	supported, value := doesServerSupportCompression(request_headers)
	if supported {
		compressed_response := CompressWithGzip(content)

		content_encoding_header := fmt.Sprintf(content_encoding_formatter, value)
		content_length_header := fmt.Sprintf(content_length_formatter, len(compressed_response))
		response_headers := fmt.Sprintf("%s%s%s", content_type_header, content_length_header, content_encoding_header)

		response_without_body := fmt.Appendf(nil, "%s\r\n%s\r\n", response_line, response_headers)
		response := append(response_without_body, compressed_response...)
		return response
	}
	content_length_header := fmt.Sprintf(content_length_formatter, len(content))
	response_headers := fmt.Sprintf("%s%s", content_type_header, content_length_header)

	return fmt.Appendf(nil, "%s\r\n%s\r\n%s", response_line, response_headers, content)
}

// doesServerSupportCompression checks the request header Accept-Encoding and returns true with supported compression, if server supports it
func doesServerSupportCompression(request_headers map[string]string) (bool, string) {
	values, exists := request_headers["Accept-Encoding"]
	homogenized_values := strings.ReplaceAll(values, ", ", ",")
	client_encodings := strings.Split(homogenized_values, ",")

	if exists {
		for i := range accepted_compression {
			for j := range client_encodings {
				if client_encodings[j] == accepted_compression[i] {
					return true, client_encodings[j]
				}
			}
		}
		return false, ""
	} else {
		return false, ""
	}
}

// CompressWithGzip takes in content and returns compressed data and its length.
func CompressWithGzip(content []byte) []byte {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(content)
	if err != nil {
		fmt.Println("Error while compressing with gzip: ", err.Error())
		return nil
	}

	err = writer.Close()
	if err != nil {
		fmt.Println("Error while closing gzip writer:", err.Error())
		return nil
	}

	return buf.Bytes()
}
