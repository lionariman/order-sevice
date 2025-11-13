package main

import (
	"testing"
)

func TestGenOrderValidates(t *testing.T) {
	o := genOrder()
	if err := o.Validate(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}
