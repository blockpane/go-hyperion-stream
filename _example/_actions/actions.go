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
