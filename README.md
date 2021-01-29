# go-hyperion-stream

[![Go Reference](https://pkg.go.dev/badge/github.com/blockpane/go-hyperion-stream.svg)](https://pkg.go.dev/github.com/blockpane/go-hyperion-stream)
[![Gosec](https://github.com/blockpane/go-hyperion-stream/workflows/Gosec/badge.svg)](https://github.com/blockpane/go-hyperion-stream/actions?query=workflow%3AGosec)
[![Build Status](https://github.com/blockpane/go-hyperion-stream/workflows/Tests/badge.svg)](https://github.com/blockpane/go-hyperion-stream/actions?workflow=Tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/blockpane/go-hyperion-stream)](https://goreportcard.com/report/github.com/blockpane/go-hyperion-stream)
[![Coverage Status](https://coveralls.io/repos/github/blockpane/go-hyperion-stream/badge.svg?branch=develop)](https://coveralls.io/github/blockpane/go-hyperion-stream?branch=develop)

This is a (minimal) library for the [Hyperion Stream API](https://hyperion.docs.eosrio.io/stream_client/).

### Example

Prints a stream of rewards paid for players of [Alien Worlds](https://alienworlds.io) on WAX

```go
package main

import (
	stream "github.com/blockpane/go-hyperion-stream"
	"log"
)

var (
	url      = "ws://wax.eosusa.news"
	contract = "m.federation"
	action   = "logmine"
	account  = ""
)

func main() {
	results := make(chan stream.HyperionResponse)
	errors := make(chan error)

	client, err := stream.NewClient(url, results, errors)
	if err != nil {
		panic(err)
	}

	err = client.StreamActions(stream.NewActionsReq(contract, account, action))
	if err != nil {
		panic(err)
	}

	act := &stream.ActionTrace{}
	for {
		select {
		case <-client.Ctx.Done():
			return
			
		case e := <-errors:
			switch e.(type) {
			case stream.ExitError:
				panic(e)
			default:
				log.Println(e)
			}

		case response := <-results:
			switch response.Type() {
			case stream.RespActionType:
				act, err = response.Action()
				if err != nil {
					log.Println(err)
					continue
				}
				log.Printf("%13s <- %11v %-13s - %v\n", act.Act.Data["miner"], act.Act.Data["bounty"], act.Act.Data["planet_name"], act.Act.Data["land_id"])
			}
		}
	}
}
```

Outputs:

```text
2021/01/28 14:06:21     pgqqy.wam <-  1.4961 TLM eyeke.world   - 1099512960814
2021/01/28 14:06:21     mnk4g.wam <-  1.0674 TLM neri.world    - 1099512958948
2021/01/28 14:06:21     v5bqy.wam <-  1.6872 TLM magor.world   - 1099512960536
2021/01/28 14:06:22     a.nqy.wam <-  0.1773 TLM kavian.world  - 1099512961065
2021/01/28 14:06:22     jgxqy.wam <-  0.9895 TLM eyeke.world   - 1099512961378
2021/01/28 14:06:22     ppjay.wam <-  2.4352 TLM magor.world   - 1099512959814
2021/01/28 14:06:22     i5jay.wam <-  0.7850 TLM kavian.world  - 1099512959254
2021/01/28 14:06:22     r53qy.wam <-  0.6678 TLM kavian.world  - 1099512958463
2021/01/28 14:06:23     x1gay.wam <-  0.4707 TLM neri.world    - 1099512958632
...
```