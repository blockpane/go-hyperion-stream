package stream

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math"
	"net/http"
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

func NewClient(url string, insecureTls bool, results chan HyperionResponse, errors chan error) (*Client, error) {
	c := &Client{}
	c.Ctx, c.cancel = context.WithCancel(context.Background())

	dial := websocket.DefaultDialer
	dial.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecureTls}
	url = strings.TrimRight(url, "/")
	conn, _, err := dial.DialContext(c.Ctx, url+`/socket.io/?EIO=3&transport=websocket`, Header)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	c.reqQueue = make(chan []byte)
	c.wait = make(chan interface{})

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	var sendPing bool
	go func() {
		ping := time.NewTicker((pongWait * 2) / 3)
		for {
			select {
			case <-c.Ctx.Done():
				errors <- ExitError{}
				// socket.io specific disconnect message:
				_ = c.conn.WriteMessage(websocket.TextMessage, []byte("41"))
				_ = c.conn.Close()
				c.cancel()
				return
			case <-ping.C:
				sendPing = true
			}
		}
	}()

	go func() {
		<-c.wait
		for {
			// FIXME: DEADLOCK!
			if sendPing {
				sendPing = false
				//err := c.WriteControl(PongMessage, []byte(message), time.Now().Add(writeWait))
				err = c.conn.WriteJSON(2)
				time.Sleep(100*time.Millisecond)
				if err != nil {
					errors <-err
				}
			}
			mtype, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					errors <- err
					c.cancel()
				}
				break
			}
			switch mtype {
			case websocket.TextMessage:
				break
			case websocket.CloseMessage:
				c.cancel()
			default:
				continue
			}

			if string(message) == "3" {
				// handle socket.io's strange nonstandard ping/pong:
				_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
				return
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
	err = c.conn.WriteMessage(websocket.TextMessage, append(append([]byte(`420["action_stream_request",`), j...), []byte("]")...))
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
