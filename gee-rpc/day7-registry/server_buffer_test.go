package geerpc

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"geerpc/codec"
	"net"
	"testing"
	"time"
)

func TestServeConnPreservesBufferedRequest(t *testing.T) {
	server := NewServer()
	var foo Foo
	if err := server.Register(&foo); err != nil {
		t.Fatal(err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	go server.ServeConn(serverConn)

	var request bytes.Buffer
	if err := json.NewEncoder(&request).Encode(&Option{
		MagicNumber: MagicNumber,
		CodecType:   codec.GobType,
	}); err != nil {
		t.Fatal(err)
	}
	encoder := gob.NewEncoder(&request)
	if err := encoder.Encode(&codec.Header{ServiceMethod: "Foo.Sum", Seq: 1}); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Encode(Args{Num1: 1, Num2: 2}); err != nil {
		t.Fatal(err)
	}

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := clientConn.Write(request.Bytes()); err != nil {
		t.Fatal(err)
	}

	decoder := gob.NewDecoder(clientConn)
	var header codec.Header
	if err := decoder.Decode(&header); err != nil {
		t.Fatal(err)
	}
	var reply int
	if err := decoder.Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if header.Error != "" || reply != 3 {
		t.Fatalf("unexpected response: header=%+v reply=%d", header, reply)
	}
}
