package util

import "testing"

func Test_TopDir(t *testing.T) {
	s := TopDir("/a")
	if s != "/a" {
		t.FailNow()
	}
	s = TopDir("a/b")
	if s != "a" {
		t.FailNow()
	}
}
