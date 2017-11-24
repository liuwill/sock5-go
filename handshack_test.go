package sock5

import (
	"net"
	"testing"
)

func Test_handshack(t *testing.T) {
	conn, err := net.Dial("tcp", ":10008")

	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 100)
	n, err := conn.Read(buf)

	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x05, 0x00}
	if string(buf[:n]) != string(expected) {
		t.Fatalf("Handshake protocal failure expected %s ,but got %s", expected, buf[:n])
	}
}
