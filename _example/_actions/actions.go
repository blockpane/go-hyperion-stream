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
