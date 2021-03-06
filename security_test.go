// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func TestNullSecurity(t *testing.T) {
	sec := nullSecurity{}
	if got, want := sec.Type(), NullSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	data := []byte("hello world")
	wenc := new(bytes.Buffer)
	if _, err := sec.Encrypt(wenc, data); err != nil {
		t.Fatalf("error encrypting data: %v", err)
	}

	if !bytes.Equal(wenc.Bytes(), data) {
		t.Fatalf("error encrypted data.\ngot = %q\nwant= %q\n", wenc.Bytes(), data)
	}

	wdec := new(bytes.Buffer)
	if _, err := sec.Decrypt(wdec, wenc.Bytes()); err != nil {
		t.Fatalf("error decrypting data: %v", err)
	}

	if !bytes.Equal(wdec.Bytes(), data) {
		t.Fatalf("error decrypted data.\ngot = %q\nwant= %q\n", wdec.Bytes(), data)
	}
}

func TestNullHandshakeReqRep(t *testing.T) {
	var (
		reqQuit = NewMsgString("QUIT")
		repQuit = NewMsgString("bye")
	)

	sec := nullSecurity{}
	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := "ipc://ipc-req-rep-null-sec"
	cleanUp(ep)

	req := NewReq(ctx, WithSecurity(sec))
	defer req.Close()

	rep := NewRep(ctx, WithSecurity(sec))
	defer rep.Close()

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return errors.Wrap(err, "could not listen")
		}

		msg, err := rep.Recv()
		if err != nil {
			return errors.Wrap(err, "could not recv REQ message")
		}

		if !reflect.DeepEqual(msg, reqQuit) {
			return errors.Errorf("got = %v, want = %v", msg, repQuit)
		}

		err = rep.Send(repQuit)
		if err != nil {
			return errors.Wrap(err, "could not send REP message")
		}

		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return errors.Wrap(err, "could not dial")
		}

		err = req.Send(reqQuit)
		if err != nil {
			return errors.Wrap(err, "could not send REQ message")
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatal(err)
	}
}

func cleanUp(ep string) {
	if strings.HasPrefix(ep, "ipc://") {
		os.Remove(ep[len("ipc://"):])
	}
}
