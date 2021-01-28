# go-hyperion-stream

This is a client for the [Hyperion Steam Client](https://hyperion.docs.eosrio.io/stream_client/).

This software is NOT complete, interfaces and data structures are in-flux, it is not stable, and entirely untested.

### Example

Prints a stream of rewards paid for players of [Alien Worlds](https://alienworlds.io) on WAX

```go
package main

import (
	"fmt"
	stream "github.com/blockpane/go-hyperion-stream"
	"log"
	"os"
)

var (
	url      = "ws://wax.eosusa.news"
	contract = "m.federation"
	action   = "logmine"
	account  = ""
)

func main() {
	log.SetFlags(log.Lshortfile|log.LstdFlags)
	fatal := func(e error) {
		if e != nil {
			_ = log.Output(2, e.Error())
			os.Exit(1)
		}
	}

	results := make(chan stream.HyperionResponse)
	errors := make(chan error)

	client, err := stream.NewClient(url, results, errors)
	fatal(err)

	err = client.StreamActions(stream.NewActionsReq(contract, account, action))
	fatal(err)

	for {
		select {
		case <-client.Ctx.Done():
			return
		case e := <-errors:
			switch e.(type) {
			case stream.ExitError:
				fatal(e)
			default:
				log.Println(e)
			}
		case response := <-results:
			switch response.Type() {
			case stream.RespActionType:
				action, err := response.Action()
				if err != nil {
					log.Println(err)
					continue
				}
				fmt.Printf("%13s <- %11v %-13s - %v\n", action.Act.Data["miner"], action.Act.Data["bounty"], action.Act.Data["planet_name"], action.Act.Data["land_id"])
			}
		}
	}
}
```
