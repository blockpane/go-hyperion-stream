package main

import (
	"fmt"
	stream "github.com/blockpane/go-hyperion-stream"
	"log"
)

var (
	url   = "ws://wax.eosrio.io"
	code  = "m.federation"
	table = "bags"
	scope = "m.federation"
	payer = ""
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

	// This sends a request to start streaming table deltas.
	//    See also: stream.NewDeltasReqByBlock and stream.NewDeltasReqByTime.
	//
	// This library imposes an arbitrary limit of one subscription per client. If another request has already been
	// sent on this client, it will return an error.
	err = client.StreamDeltas(stream.NewDeltasReq(code, table, scope, payer))
	if err != nil {
		panic(err)
	}

	delta := &stream.DeltaTrace{}
	for {
		select {

		// client.Ctx is a context that will close when the websocket is torn down.
		case <-client.Ctx.Done():
			return

		// any errors that occur will come over this channel.
		case e := <-errors:
			switch e.(type) {
			// stream.ExitError will be sent when the non-standard socket.io disconnect message is received.
			case stream.ExitError:
				panic(e)
			default:
				log.Println(e)
			}

		// a message was received
		case response := <-results:
			// stream.HyperionResponse is an interface with multiple types, first we check the type to ensure it's
			// a stream.DeltaTrace.
			switch response.Type() {
			case stream.RespDeltaType:
				// the Delta() function will return a *stream.DeltaTrace, if this is not a delta it will error.
				delta, err = response.Delta()
				if err != nil {
					log.Println(err)
					continue
				}
				// the Data field will normally be a map[string]interface{} mirroring the JSON in the trace.
				fmt.Printf("%+v\n", delta.Data)
			}
		}
	}
}
