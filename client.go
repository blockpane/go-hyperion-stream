package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"nhooyr.io/websocket"
	"strings"
	"time"
)

var (
	Header = http.Header{"User-Agent": {"go-hyperion-stream"}} // allow user-agent override
)

const (
	pongWait       = 25 * time.Second
	maxMessageSize = 8192
)

type Client struct {
	Ctx context.Context

	LibNum  uint32
	LibId   string
	ChainId string

	conn       *websocket.Conn
	reqQueue   chan []byte
	cancel     func()
	subscribed bool
	wait       chan interface{}
}

func NewClient(url string, results chan HyperionResponse, errors chan error) (*Client, error) {
	c := &Client{}
	c.Ctx, c.cancel = context.WithCancel(context.Background())

	url = strings.TrimRight(url, "/")
	conn, _, err := websocket.Dial(c.Ctx, url+`/socket.io/?EIO=3&transport=websocket`, &websocket.DialOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		return nil, err
	}
	c.conn = conn
	c.reqQueue = make(chan []byte)
	c.wait = make(chan interface{})
	c.conn.SetReadLimit(maxMessageSize)

	go func() {
		ping := time.NewTicker((pongWait * 2) / 3)
		for {
			select {
			case <-c.Ctx.Done():
				errors <- ExitError{}
				// socket.io specific disconnect message:
				_ = c.conn.Write(c.Ctx, websocket.MessageText, []byte("41"))
				_ = c.conn.CloseRead(context.Background())
				c.cancel()
				return
			case <-ping.C:
				err = c.conn.Write(c.Ctx, websocket.MessageText, []byte("2"))
				if err != nil {
					errors <-err
				}
			}
		}
	}()

	go func() {
		<-c.wait
		for {
			mtype, message, err := c.conn.Read(c.Ctx)
			if err != nil {
				errors <-err
				c.cancel()
				break
			}
			switch true {
			case mtype == websocket.MessageText:
				break
			case mtype > 1000:
				c.cancel()
			default:
				continue
			}

			if len(message) < 2 || string(message[:2]) != "42" {
				// only care about event messages from here:
				continue
			}

			go func(m []byte) {
				raw := make([]interface{}, 2)
				err = json.Unmarshal(m[2:], &raw)
				if err != nil {
					errors <- err
					return
				}

				// if it's not a string we're going to end up in trouble since reflection is involved, some socket.io
				// status messages pass arrays or objects here.
				switch raw[0].(type) {
				case string:
					break
				default:
					return
				}

				switch raw[0].(string) {
				case "lib_update":
					// track lib updates in stream.Client variables:
					update := raw[1].(map[string]interface{})
					if update["chain_id"] == nil || update["block_num"] == nil || update["block_id"] == nil {
						return
					}
					c.ChainId = update["chain_id"].(string)
					c.LibNum = uint32(math.Round(update["block_num"].(float64)))
					c.LibId = update["block_id"].(string)
					return
				case "message":
					break
				default:
					// everything else we don't care
					return
				}

				// make sure we have have a map
				switch raw[1].(type) {
				case map[string]interface{}:
					break
				default:
					return
				}

				if raw[1].(map[string]interface{})["type"] == nil || raw[1].(map[string]interface{})["message"] == nil {
					return
				}

				switch raw[1].(map[string]interface{})["type"].(string) {
				case "delta":
					// TODO, and btw is it even "delta"???
				case "action_trace":
					a := &ActionTrace{}
					//fmt.Println(raw[1].(map[string]interface{})["message"].(string))
					err = json.Unmarshal([]byte(raw[1].(map[string]interface{})["message"].(string)), a)
					if err != nil {
						errors <-err
						return
					}
					//j, _ := json.MarshalIndent(a, "", "  ")
					//fmt.Println(string(j))
					results <-a
				// FIXME: delete debug cruft
				default:
					fmt.Println("unknown message:", raw[1].(map[string]interface{})["type"].(string))
				}
			}(message)
		}
	}()

	return c, err
}

func (c *Client) StreamActions(req *ActionsReq) error {
	if c.subscribed {
		return BusyError{}
	}
	j, err := req.ToJson()
	if err != nil {
		return err
	}
	err = c.conn.Write(c.Ctx, websocket.MessageText, append(append([]byte(`420["action_stream_request",`), j...), []byte("]")...))
	if err != nil {
		return err
	}
	c.subscribed = true
	close(c.wait)
	return nil
}

type ExitError struct{}

func (e ExitError) Error() string {
	return "websocket closed"
}

type BusyError struct{}

func (b BusyError) Error() string {
	return "websocket subscription already active, please use a new client for additional subscriptions"
}
