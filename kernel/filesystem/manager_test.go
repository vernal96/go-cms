package filesystem

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type testFactory struct {
	code filesystemCode
	disk *testDisk
	err  error
}

type filesystemCode = Code

func (f testFactory) Code() Code { return Code(f.code) }
func (f testFactory) Open(context.Context) (Disk, error) {
	return f.disk, f.err
}

type testDisk struct {
	code       Code
	visibility Visibility
	pings      atomic.Int32
	closes     atomic.Int32
}

func (d *testDisk) Code() Code                 { return d.code }
func (d *testDisk) Visibility() Visibility     { return d.visibility }
func (d *testDisk) Ping(context.Context) error { d.pings.Add(1); return nil }
func (d *testDisk) Close() error               { d.closes.Add(1); return nil }
func (d *testDisk) PutNew(context.Context, string, io.Reader, string) error {
	return nil
}
func (d *testDisk) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, ErrNotFound
}
func (d *testDisk) Delete(context.Context, string) error { return nil }
func (d *testDisk) URL(context.Context, Reference) (string, error) {
	return "", nil
}
func (d *testDisk) TemporaryURL(
	context.Context,
	Reference,
	time.Time,
) (string, error) {
	return "", nil
}

func TestManagerOpensResolvesAndClosesDisks(t *testing.T) {
	public := &testDisk{code: "public", visibility: VisibilityPublic}
	private := &testDisk{code: "private", visibility: VisibilityPrivate}

	manager, err := NewManager(context.Background(), []Factory{
		testFactory{code: "public", disk: public},
		testFactory{code: "private", disk: private},
	})
	if err != nil {
		t.Fatal(err)
	}
	if disk, exists := manager.Disk("private"); !exists || disk != private {
		t.Fatalf("private disk = %#v, %t", disk, exists)
	}
	if public.pings.Load() != 1 || private.pings.Load() != 1 {
		t.Fatalf("ping counts = %d, %d", public.pings.Load(), private.pings.Load())
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if public.closes.Load() != 1 || private.closes.Load() != 1 {
		t.Fatalf("close counts = %d, %d", public.closes.Load(), private.closes.Load())
	}
}

func TestManagerRejectsDuplicatesAndClosesPartialOpen(t *testing.T) {
	opened := &testDisk{code: "public", visibility: VisibilityPublic}
	_, err := NewManager(context.Background(), []Factory{
		testFactory{code: "public", disk: opened},
		testFactory{code: "broken", err: errors.New("open failed")},
	})
	if err == nil {
		t.Fatal("expected open error")
	}
	if opened.closes.Load() != 1 {
		t.Fatalf("partial disk close count = %d", opened.closes.Load())
	}

	_, err = NewManager(context.Background(), []Factory{
		testFactory{
			code: "same",
			disk: &testDisk{code: "same", visibility: VisibilityPublic},
		},
		testFactory{
			code: "same",
			disk: &testDisk{code: "same", visibility: VisibilityPrivate},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate disk error")
	}
}
