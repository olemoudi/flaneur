package main

import "testing"

func TestTrimMD5(t *testing.T) {

	exp := "4d1"
	act := trimMD5("hola", 3)

	if exp != act {
		t.Fatalf("Expected %s but got %s", exp, act)
	}

	exp = "4d186321"
	act = trimMD5("hola", 8)

	if exp != act {
		t.Fatalf("Expected %s but got %s", exp, act)
	}

}
