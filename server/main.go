package main

import (
	"crypto/sha1"
	"crypto/subtle"
	"flag"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	fileF := flag.String("file", "/usr/bin/emacs", "file to compare with received")
	flag.Parse()

	f, err := os.Open(*fileF)
	if err != nil {
		log.Fatal(err)
	}

	// calculate expected hash
	hash := sha1.New()
	_, err = io.Copy(hash, f)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	expected := hash.Sum(nil)
	log.Printf("expected hash %40x", expected)
	hash.Reset()

	// listen and accept single connection
	l, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := l.Accept()
	if err != nil {
		log.Fatal(err)
	}

	var done bool
	var reads, bytes, timeouts int
	for !done {
		timeout := time.Duration(rand.Intn(200)) * time.Microsecond
		toRead := rand.Intn(1 * 1024 * 1024)

		err = conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Fatal(err)
		}

		// read data
		b := make([]byte, toRead)
		reads++
		n, err := io.ReadFull(conn, b)
		bytes += n
		if n > 0 && err != nil {
			log.Printf("read %d bytes, wanted to read %d, %s", n, len(b), err)
		}

		_, _ = hash.Write(b[:n]) // hash.Hash.Write never returns errors

		// check if we are done
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			done = true
			err = nil
		}

		// count timeouts
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			timeouts++
			err = nil
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("%d reads, %d bytes, %d timeouts.", reads, bytes, timeouts)

	actual := hash.Sum(nil)
	log.Printf("expected %40x, got %40x", expected, actual)
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		log.Fatal("data corrupted")
	}
}
