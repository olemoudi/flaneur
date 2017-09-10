package main

import "testing"

func TestMD5Prefix(t *testing.T) {

	exp := "4d1"
	act := MD5Prefix("hola", 3)

	if exp != act {
		t.Fatalf("Expected %s but got %s", exp, act)
	}

	exp = "4d186321"
	act = MD5Prefix("hola", 8)

	if exp != act {
		t.Fatalf("Expected %s but got %s", exp, act)
	}

}
