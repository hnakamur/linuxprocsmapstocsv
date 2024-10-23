package main

import (
	"strings"
	"testing"
)

func TestParseRegion(t *testing.T) {
	r, err := parseRegion([]byte("4d400283000-4d400284000 ---p 00000000 00:00 0                            [anon:partition_alloc]"))
	if err != nil {
		t.Fatal(err)
	}
	m := mapping{Region: r}
	if got, want := strings.Join(m.toCSVRecord(), ","), "4d400283000,4d400284000,---p,00000000,00:00,0,[anon:partition_alloc]"; got != want {
		t.Errorf("result mismatch,\n got=%s,\nwant=%s", got, want)
	}
}
