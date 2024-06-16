package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strings"
)

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

		go handleConnections(conn)
	}
}

func handleConnections(conn net.Conn) {
	defer conn.Close()

	requestBuffer := make([]byte, 1024)

	n, err := conn.Read(requestBuffer)
	if err != nil {
		fmt.Println("Failed to read request: ", err)
		return
	}
	fmt.Printf("Request: %s\n", requestBuffer[:n])

	request := string(requestBuffer[:n])
	method := strings.Split(request, " ")[0]
	path := strings.Split(request, " ")[1]

	if method == "GET" {
		if path == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.Split(path, "/")[1] == "echo" {
			message := strings.Split(path, "/")[2]

			for _, line := range strings.Split(request, "\n") {
				if strings.HasPrefix(line, "Accept-Encoding:") {
					encoding := strings.Split(strings.TrimSpace(line), ":")[1]
					requestEncodingList := strings.Join(strings.Split(strings.TrimSpace(encoding), ", "), " ")

					if strings.Contains(requestEncodingList, "gzip") {
						var buffer bytes.Buffer

						gzipWriter := gzip.NewWriter(&buffer)
						_, err := gzipWriter.Write([]byte(message))
						if err != nil {
							fmt.Println("Error compressing message: ", err)
							return
						}
						if err := gzipWriter.Close(); err != nil {
							fmt.Println("Error flushing compressed message:", err)
							return
						}

						body := buffer.String()

						conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", "gzip", len(body), body)))
						return
					}
				}
			}

			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)))
		} else if strings.Split(path, "/")[1] == "user-agent" {
			userAgent := strings.TrimSpace(strings.Split(request, "User-Agent:")[1])
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)))
		} else if strings.Split(path, "/")[1] == "files" {
			requestedFile := strings.Split(path, "/")[2]
			dir := os.Args[2]
			content, err := os.ReadFile(dir + requestedFile)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				return
			}
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(string(content)), string(content))))
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	} else if method == "POST" {
		if strings.Split(path, "/")[1] == "files" {
			requestedFile := strings.Split(path, "/")[2]
			dir := os.Args[2]

			lines := strings.Split(request, "\n")
			content := lines[len(lines)-1]

			os.WriteFile(dir+requestedFile, []byte(content), 0644)

			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
		}
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
