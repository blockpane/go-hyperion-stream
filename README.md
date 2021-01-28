# go-hyperion-stream

This is a library for the [Hyperion Stream API](https://hyperion.docs.eosrio.io/stream_client/).

This software is NOT complete, interfaces and data structures are in-flux, it is not stable, and entirely untested.

## Todo:

- implement delta events
- better error handling
- tests

### Example

Prints a stream of rewards paid for players of [Alien Worlds](https://alienworlds.io) on WAX

```go
package main

import (
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
				log.Printf("%13s <- %11v %-13s - %v\n", action.Act.Data["miner"], action.Act.Data["bounty"], action.Act.Data["planet_name"], action.Act.Data["land_id"])
			}
		}
	}
}
```

Outputs:

```text
2021/01/28 00:20:22 actions.go:53:     t1sqw.wam <-  0.7808 TLM neri.world    - 1099512960086
2021/01/28 00:20:23 actions.go:53:     x5fqy.wam <-  0.8697 TLM neri.world    - 1099512960946
2021/01/28 00:20:23 actions.go:53:     pwway.wam <-  1.9841 TLM naron.world   - 1099512961215
2021/01/28 00:20:23 actions.go:53:     t4vay.wam <-  0.9841 TLM kavian.world  - 1099512961373
2021/01/28 00:20:24 actions.go:53:     lkxqy.wam <-  0.9859 TLM kavian.world  - 1099512961373
2021/01/28 00:20:24 actions.go:53:     eswqy.wam <-  8.6215 TLM magor.world   - 1099512961402
...
```