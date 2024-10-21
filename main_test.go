package main

import "testing"

func TestParseRegion(t *testing.T) {
	r, err := parseRegion([]byte("4d400283000-4d400284000 ---p 00000000 00:00 0                            [anon:partition_alloc]"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.String(), "4d400283000,4d400284000,---p,00000000,00:00,0,[anon:partition_alloc]"; got != want {
		t.Errorf("result mismatch,\n got=%s,\nwant=%s", got, want)
	}
}
