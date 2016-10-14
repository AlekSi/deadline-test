package main

import (
	"crypto/sha1"
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
	fileF := flag.String("file", "/usr/bin/emacs", "file to send")
	flag.Parse()

	f, err := os.Open(*fileF)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		log.Fatal(err)
	}

	hash := sha1.New()

	var done bool
	var writes, bytes, timeouts int
	for !done {
		timeout := time.Duration(rand.Intn(200)) * time.Microsecond
		toWrite := rand.Intn(1 * 1024 * 1024)

		err = conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Fatal(err)
		}

		// read data from file
		b := make([]byte, toWrite)
		n, err := io.ReadFull(f, b)
		b = b[:n]

		// check if we are done
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			done = true
			err = nil
		}
		if err != nil {
			log.Fatal(err)
		}

		// write data
		writes++
		n, err = conn.Write(b)
		bytes += n
		if n > 0 && err != nil {
			log.Printf("wrote %d bytes, wanted to write %d bytes, %s", n, len(b), err)
		}

		_, _ = hash.Write(b[:n]) // hash.Hash.Write never returns errors

		// count timeouts
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			timeouts++
			err = nil
		}
		if err != nil {
			log.Fatal(err)
		}

		// rewind file position
		offset := int64(n - len(b))
		if offset != 0 {
			_, err = f.Seek(offset, io.SeekCurrent)
			if err != nil {
				log.Fatal(err)
			}

			// we are _not_ done if we need to rewind
			done = false
		}
	}

	conn.Close()
	log.Printf("%d writes, %d bytes, %d timeouts.", writes, bytes, timeouts)
	log.Printf("expected hash %40x", hash.Sum(nil))
}
