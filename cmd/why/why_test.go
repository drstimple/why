package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWhyWhy(t *testing.T) {
	var b bytes.Buffer
	printWhyWhy(&b, false)
	for _, want := range []string{"WHY DOES WHY EXIST?", "Two nerds thought it would be fun.", "None. Not for profit.", "Entirely reasonable."} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("missing %q: %s", want, b.String())
		}
	}
	b.Reset()
	printWhyWhy(&b, true)
	var obj map[string]string
	if err := json.Unmarshal(b.Bytes(), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["verdict"] != "Entirely reasonable." {
		t.Fatal(obj)
	}
}

func TestRecursiveWhy(t *testing.T) {
	args := []string{"--why", "--why", "--why", "--json"}
	if countWhyFlags(args) != 3 {
		t.Fatal("wrong depth")
	}
	normalized := normalizeWhyFlags(args)
	if strings.Join(normalized, " ") != "--why --json" {
		t.Fatalf("unexpected args: %v", normalized)
	}
	for _, jsonMode := range []bool{false, true} {
		var b bytes.Buffer
		printRecursiveWhy(&b, jsonMode)
		if jsonMode {
			var result map[string]string
			if err := json.Unmarshal(b.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result["verdict"] != "Go outside." {
				t.Fatal(result)
			}
		} else if !strings.Contains(b.String(), "RECURSIVE EXISTENTIAL INQUIRY DETECTED") || !strings.Contains(b.String(), "Go outside.") {
			t.Fatal(b.String())
		}
	}
}
