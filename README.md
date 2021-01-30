# go-hyperion-stream

[![Go Reference](https://pkg.go.dev/badge/github.com/blockpane/go-hyperion-stream.svg)](https://pkg.go.dev/github.com/blockpane/go-hyperion-stream)
[![Gosec](https://github.com/blockpane/go-hyperion-stream/workflows/Gosec/badge.svg)](https://github.com/blockpane/go-hyperion-stream/actions?query=workflow%3AGosec)
[![CodeQL](https://github.com/blockpane/go-hyperion-stream/workflows/CodeQL/badge.svg)](https://github.com/blockpane/go-hyperion-stream/actions?query=workflow%3ACodeQL)
[![Build Status](https://github.com/blockpane/go-hyperion-stream/workflows/Tests/badge.svg)](https://github.com/blockpane/go-hyperion-stream/actions?workflow=Tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/blockpane/go-hyperion-stream)](https://goreportcard.com/report/github.com/blockpane/go-hyperion-stream)
[![Coverage Status](https://coveralls.io/repos/github/blockpane/go-hyperion-stream/badge.svg?branch=develop)](https://coveralls.io/github/blockpane/go-hyperion-stream?branch=develop)

This is a (minimal) library for the [Hyperion Stream API](https://hyperion.docs.eosrio.io/stream_client/).

### Example

Prints a stream of rewards paid to players of [Alien Worlds](https://alienworlds.io) on WAX

```go
package main

import (
	stream "github.com/blockpane/go-hyperion-stream"
	"log"
)

var (
	url      = "wss://wax.eosrio.io"
	contract = "m.federation"
	action   = "logmine"
	account  = ""
)

func main() {
	// create two channels, the first receives HyperionResponse interfaces, the second any errors.
	results := make(chan stream.HyperionResponse)
	errors := make(chan error)

	// NewClient will immediately connect, in the background it constantly updates client.LibId with the highest
	// irreversible block, and handles socket.io's unique client-initiated ping/pong sequences.
	client, err := stream.NewClient(url, results, errors)
	if err != nil {
		panic(err)
	}

	// This sends a request to start streaming actions.
	//    See also: stream.NewActionsReqByBlock and stream.NewActionsReqByTime.
	//
	// This library imposes an arbitrary limit of one subscription per client. If another request has already been
	// sent on this client, it will return an error.
	err = client.StreamActions(stream.NewActionsReq(contract, account, action))
	if err != nil {
		panic(err)
	}

	act := &stream.ActionTrace{}
	for {
		select {

		// client.Ctx is a context that will close when the websocket is torn down.
		case <-client.Ctx.Done():
			return

		// any errors that occur will come over this channel.
		case e := <-errors:
			switch e.(type) {
			// stream.ExitError will be sent when the non-standard socket.io disconnect message is received
			case stream.ExitError:
				panic(e)
			default:
				log.Println(e)
			}

		// a message was received
		case response := <-results:
			// stream.HyperionResponse is an interface with multiple types, first we check the type to ensure it's
			// a stream.ActionTrace.
			switch response.Type() {
			case stream.RespActionType:
				// the Action() function will return a *stream.ActionTrace, if this is not an action it will error.
				act, err = response.Action()
				if err != nil {
					log.Println(err)
					continue
				}
				// the Act.Data field will normally be a map[string]interface{} mirroring the JSON in the trace.
				log.Printf("%13s <- %11v %-13s - %v\n", act.Act.Data["miner"], act.Act.Data["bounty"], act.Act.Data["planet_name"], act.Act.Data["land_id"])
			}
		}
	}
}
```

Outputs:

```text
2021/01/29 21:46:21     4hhqy.wam <-  4.2301 TLM magor.world   - 1099512960752
2021/01/29 21:46:22     ikway.wam <-  0.1104 TLM neri.world    - 1099512960086
2021/01/29 21:46:22     fgnqy.wam <-  0.1582 TLM kavian.world  - 1099512961377
2021/01/29 21:46:22     1gway.wam <-  0.0610 TLM kavian.world  - 1099512961373
2021/01/29 21:46:22     14vqy.wam <-  0.1382 TLM veles.world   - 1099512960189
2021/01/29 21:46:22     jcmqy.wam <-  1.0106 TLM naron.world   - 1099512960233
2021/01/29 21:46:23     w4zay.wam <-  1.0831 TLM kavian.world  - 1099512958608
2021/01/29 21:46:23     5hoay.wam <-  0.2749 TLM neri.world    - 1099512960946
. . .
```
