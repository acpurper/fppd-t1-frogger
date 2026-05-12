package main

import "testing"

func TestParseInputByteArrowSequences(t *testing.T) {
	var state int
	sequence := []byte{0x1b, '[', 'A'}
	for i, b := range sequence {
		cmd, ok := parseInputByte(&state, b)
		if i < len(sequence)-1 {
			if ok {
				t.Fatalf("unexpected command on byte %d", i)
			}
			continue
		}
		if !ok || cmd != CmdUp {
			t.Fatalf("arrow up sequence did not parse correctly: got %v %v", cmd, ok)
		}
	}
}

func TestParseInputByteWindowsArrowCodes(t *testing.T) {
	var state int
	sequence := []byte{0x00, 72}
	for i, b := range sequence {
		cmd, ok := parseInputByte(&state, b)
		if i < len(sequence)-1 {
			if ok {
				t.Fatalf("unexpected command on byte %d", i)
			}
			continue
		}
		if !ok || cmd != CmdUp {
			t.Fatalf("windows arrow up code did not parse correctly: got %v %v", cmd, ok)
		}
	}
}
