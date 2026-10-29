package main

import "testing"

func TestSum(t *testing.T) {
    if sum(2, 3) != 5 {
        t.Fatal("expected sum(2, 3) == 5")
    }
}
