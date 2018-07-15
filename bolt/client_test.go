package bolt

import (
	"os"
	"reflect"
	"testing"
)

func TestClient(t *testing.T) {
	client, err := New()
	defer os.Remove("my.db")
	if err != nil {
		t.Error(err)
		return
	}
	a := A{
		P1: "aaaaa",
		P2: 42,
		P3: true,
	}
	if err := client.Update(PhenomeBucket, []byte("key"), &a); err != nil {
		t.Error(err)
		return
	}
	b := new(A)
	if err := client.Get(PhenomeBucket, []byte("key"), b); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(a, *b) {
		t.Errorf("a & b are not equal: %+v, %+v", a, *b)
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

type A struct {
	P1 string
	P2 int
	P3 bool
}
