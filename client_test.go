package stream

import (
	"context"
	"log"
	"net/http"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"testing"
	"time"
)

func wsDriver() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
		defer cancel()

		var v interface{}
		for {
			err = wsjson.Read(ctx, c, &v)
			if err != nil {
				break
			}
		}
		_ = c.Close(websocket.StatusNormalClosure, "")
	})
	http.HandleFunc("/socket.io/", handler)
	_ = http.ListenAndServe("localhost:23456", nil)
}

func TestStreamActions(t *testing.T) {
	results := make(chan HyperionResponse)
	errors := make(chan error)
	go wsDriver()
	c, err := NewClient("ws://localhost:23456", results, errors)
	if err != nil {
		t.Error("new client:", err)
	}

	err = c.StreamActions(NewActionsReq("a", "b", "c"))
	if err != nil {
		t.Error("stream actions:", err)
	}

	for {
		select {
		case <-time.After(time.Second):
			c.cancel()
		case <-c.Ctx.Done():
			return
		case m := <-results:
			log.Println(m)
		case <-errors:
			// noop
		}
	}
}

func TestStreamDeltas(t *testing.T) {
	results := make(chan HyperionResponse)
	errors := make(chan error)
	c, err := NewClient("ws://localhost:23456", results, errors)
	if err != nil {
		t.Error("new client:", err)
	}

	err = c.StreamDeltas(NewDeltasReq("a", "b", "c", ""))
	if err != nil {
		t.Error("stream deltas:", err)
	}

	for {
		select {
		case <-time.After(time.Second):
			c.cancel()
		case <-c.Ctx.Done():
			return
		case m := <-results:
			log.Println(m)
		case <-errors:
			// noop
		}
	}
}

func TestClientErrors(t *testing.T) {
	switch "" {
	case ExitError{}.Error():
		t.Error("ExitError empty")
		fallthrough
	case BusyError{}.Error():
		t.Error("ExitError empty")
	}
}
