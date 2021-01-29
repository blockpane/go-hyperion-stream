package stream

import (
	"context"
	"testing"
	"time"
)

func TestNewActionsReq(t *testing.T) {
	if r := NewActionsReq("a", "b", "c"); r == nil {
		t.Error("got nil request")
	}
	if r := NewActionsReqByBlock("a", "b", "c", 0, 0); r == nil {
		t.Error("got nil request")
	}
	if r := NewActionsReqByBlock("a", "b", "c", 1, 0); r == nil {
		t.Error("got nil request")
	}
	if r := NewActionsReqByTime("a", "b", "c", time.Now().Format(time.RFC3339), ""); r == nil {
		t.Error("got nil request")
	} else {
		if ok := r.AddFilter(&ReqFilter{Field: "", Value: ""}); !ok {
			t.Error("could not add filter")
		}
		if ok := r.AddFilter(nil); ok {
			t.Error("should not be able to supply nil filter")
		}
		b, e := r.ToJson()
		if e != nil {
			t.Error(e)
		}
		if b == nil || len(b) < 10 {
			t.Error("json too small")
		}
	}
}

func TestNewDeltaReq(t *testing.T) {
	// TODO: these simply ensure coverage
	if r := NewDeltasReq("a", "b", "c", ""); r == nil {
		t.Error("got nil request")
	}
	if r := NewDeltasReqByBlock("a", "b", "c", "", 1, 0); r == nil {
		t.Error("got nil request")
	}
	if r := NewDeltasReqByBlock("a", "b", "c", "", 0, 0); r == nil {
		t.Error("got nil request")
	}
	if r := NewDeltasReqByTime("a", "b", "c", "", time.Now().Format(time.RFC3339), ""); r == nil {
		t.Error("got nil request")
	}
	if r := NewDeltasReqByTime("a", "b", "c", "", "", ""); r == nil {
		t.Error("got nil request")
	} else {
		b, e := r.ToJson()
		if e != nil {
			t.Error(e)
		}
		if b == nil || len(b) < 10 {
			t.Error("invalid json")
		}
	}
}

func TestDeltaMsg(t *testing.T) {
	const (
		deltaTraceMessage = `42["message",{"type":"delta_trace","mode":"live","message":"{\"code\":\"m.federation\",\"scope\":\"m.federation\",\"table\":\"bags\",\"primary_key\":\"16158474573985087488\",\"payer\":\"w.zay.wam\",\"@timestamp\":\"2021-01-28T19:03:01.000\",\"present\":true,\"block_num\":100851918,\"block_id\":\"0602e0cee78f6880ba083cc4781ee31b0998ce752ccf2bc348beef555e0a1f1f\",\"data\":{\"account\":\"w.zay.wam\",\"items\":[\"1099513962800\",\"1099513883909\",\"1099514157356\"],\"locked\":false}}"}]`
	)

	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan HyperionResponse)
	errors := make(chan error)
	client := &Client{}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errors:
				t.Error(err)
			case message := <-results:
				switch message.Type() {
				case RespDeltaType:
					break
				default:
					t.Error("did not receive delta message from HyperionResponse channel")
					cancel()
					return
				}

				if d, e := message.Action(); e == nil || d != nil {
					t.Error("delta trace cannot convert to action")
				}

				delt, err := message.Delta()
				if err != nil {
					t.Error(err)
				}
				if delt == nil {
					t.Error("nil delta on self cast")
					cancel()
				}

				b := delt.ToJson()
				if b == nil {
					t.Error("got nil json")
				}
				// arbitrary size, but ensures we got something large...
				if len(b) < 256 {
					t.Error("json was too small")
				}
				_ = message.Mode()

				cancel()
			}
		}
	}()

	var a *DeltaTrace
	b := a.ToJson()
	if b != nil {
		t.Error("empty delta returned json")
	}

	raw, ok := getRaw([]byte(deltaTraceMessage), client, errors)
	switch true {
	case !ok:
		t.Error("delta trace did not parse correctly")
		fallthrough
	case raw == nil:
		t.Error("raw message was nil")
		return
	case len(raw) != 2:
		t.Error("incorrect size for raw message")
		return
	}

	switch raw[0].(type) {
	case string:
		if raw[0].(string) == "message" {
			break
		}
		t.Error("raw[0] type was not a message")
		return
	default:
		t.Error("raw[0] did not contain a string")
		return
	}
	sendResult(raw, results, errors)

	<-ctx.Done()
}

func TestActionMsg(t *testing.T) {
	const (
		actionTraceMessage = `42["message",{"type":"action_trace","mode":"live","message":"{\"action_ordinal\":5,\"creator_action_ordinal\":1,\"act\":{\"account\":\"m.federation\",\"name\":\"logmine\",\"authorization\":[{\"actor\":\"m.federation\",\"permission\":\"log\"}],\"data\":{\"miner\":\"sp4ay.wam\",\"params\":{\"invalid\":0,\"error\":\"\",\"delay\":340,\"difficulty\":3,\"ease\":68,\"luck\":17,\"commission\":500},\"bounty\":\"0.4048 TLM\",\"land_id\":\"1099512961385\",\"planet_name\":\"neri.world\",\"landowner\":\"ve.qu.wam\",\"bag_items\":[\"1099514303963\",\"1099514347290\",\"1099514364881\"],\"offset\":107}},\"context_free\":false,\"elapsed\":\"76\",\"@timestamp\":\"2021-01-28T19:37:19.000\",\"block_num\":100856033,\"producer\":\"cryptolions1\",\"trx_id\":\"53cdc7714dc40cc0042c45215dd48023afad51d54dcce75ddb6354c85d064888\",\"global_sequence\":957257254,\"receipts\":[{\"receiver\":\"m.federation\",\"global_sequence\":\"957257254\",\"recv_sequence\":\"36534368\",\"auth_sequence\":[{\"account\":\"m.federation\",\"sequence\":\"50649143\"}]}],\"code_sequence\":36,\"abi_sequence\":10,\"notified\":[\"m.federation\"]}"}]`
		libUpdateMessage   = `42["lib_update",{"chain_id":"1064487b3cd1a897ce03ae5b6a865651747e2e152090f99c1d19d44e01aea5a4","block_num":100855706,"block_id":"0602EF9A11985228B31A8711A2402B354DD07697EAA68A2FB772B876B11FB17E"}]`
	)

	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan HyperionResponse)
	errors := make(chan error)
	client := &Client{}
	wantError := true

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errors:
				if wantError {
					continue
				}
				t.Error(err)
			case message := <-results:
				switch message.Type() {
				case RespActionType:
					break
				default:
					t.Error("did not receive action message from HyperionResponse channel")
					cancel()
					return
				}

				if d, e := message.Delta(); e == nil || d != nil {
					t.Error("action trace cannot convert to delta")
				}

				act, err := message.Action()
				if err != nil {
					t.Error(err)
				}
				if act == nil {
					t.Error("nil action on self cast")
					cancel()
				}

				b := act.ToJson()
				if b == nil {
					t.Error("got nil json")
				}
				// arbitrary size, but ensures we got something large...
				if len(b) < 512 {
					t.Error("json was too small")
				}

				_ = message.Mode()
				cancel()
			}
		}
	}()

	var a *ActionTrace
	b := a.ToJson()
	if b != nil {
		t.Error("empty trace returned json")
	}

	// LIB update message:
	raw, ok := getRaw([]byte(libUpdateMessage), client, errors)
	switch true {
	case ok:
		t.Error("lib update should not return true")
		fallthrough
	case raw != nil:
		t.Error("lib update should not return a raw object")
		fallthrough
	case client.LibNum == 0:
		t.Error("client lib did not update")
	}
	// this *will* cause an error, it's not a trace:
	sendResult(raw, results, errors)

	raw, ok = getRaw([]byte(actionTraceMessage), client, errors)
	switch true {
	case !ok:
		t.Error("action trace did not parse correctly")
		fallthrough
	case raw == nil:
		t.Error("raw message was nil")
		return
	case len(raw) != 2:
		t.Error("incorrect size for raw message")
		return
	}

	switch raw[0].(type) {
	case string:
		if raw[0].(string) == "message" {
			break
		}
		t.Error("raw[0] type was not a message")
		return
	default:
		t.Error("raw[0] did not contain a string")
		return
	}

	// this should not cause an error, it really is a trace
	wantError = false
	sendResult(raw, results, errors)

	<-ctx.Done()
}

func Test_Error(t *testing.T) {
	switch "" {
	case NotActionError{}.Error():
		t.Error("err is empty")
		fallthrough
	case NotDeltaError{}.Error():
		t.Error("err is empty")
		fallthrough
	case UnknownTypeError{}.Error():
		t.Error("err is empty")
	}
}
