package server

import "testing"

func TestReadFrameData(t *testing.T) {
	typ, body, err := readFrameData([]byte{0x01, 'a', 'b'})
	if err != nil {
		t.Fatal(err)
	}
	if typ != 0x01 {
		t.Fatalf("expected 0x01, got %x", typ)
	}
	if string(body) != "ab" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}
