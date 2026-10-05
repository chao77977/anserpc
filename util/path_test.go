package util

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if ok, err := Exists(dir); err != nil || !ok {
		t.Fatalf("existing dir: ok=%v err=%v", ok, err)
	}

	missing := filepath.Join(dir, "nope")
	if ok, err := Exists(missing); err != nil || ok {
		t.Fatalf("missing path: ok=%v err=%v", ok, err)
	}

	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if ok, err := Exists(f); err != nil || !ok {
		t.Fatalf("existing file: ok=%v err=%v", ok, err)
	}
}

func TestMakeFilePath(t *testing.T) {
	dir := t.TempDir()

	// Parent already exists: no-op, no error.
	existing := filepath.Join(dir, "a.txt")
	if err := MakeFilePath(existing); err != nil {
		t.Fatalf("existing parent: %v", err)
	}

	// Parent must be created.
	nested := filepath.Join(dir, "sub", "b.txt")
	if err := MakeFilePath(nested); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if ok, _ := Exists(filepath.Join(dir, "sub")); !ok {
		t.Fatal("parent dir should have been created")
	}
}

func TestInterrupt_RegisterAndFire(t *testing.T) {
	// Register a callback and ensure the monitor goroutine starts without
	// panicking. We cannot easily deliver a real signal in a unit test, so
	// we invoke the internal callback dispatch directly.
	var wg sync.WaitGroup
	wg.Add(1)
	var once sync.Once
	RegisterOnInterrupt(func() { once.Do(wg.Done) })

	// nil callback is ignored.
	RegisterOnInterrupt(nil)

	// Directly exercise the fan-out path.
	_interrupter.callback()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("registered interrupt callback was not invoked")
	}
}
