package stream

import (
	"context"
	"encoding/json"
	"math"
	"nhooyr.io/websocket"
	"strings"
	"time"
)

const (
	pongWait       = 25 * time.Second
	maxMessageSize = 32768
)

// Client is a streaming client using a websocket to connect to Hyperion. The Client.Ctx will get closed when the
// websocket is terminated.
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

// NewClient immediately connects to Hyperion, handles ping/pongs, and stores state information such as last
// irreversible block number in the Client.LibNum. It expects two channels for sending results and errors.
// Once connected a query will need to be sent before any output is sent over the results channel. If no request is
// sent in the first 25 seconds the websocket will be closed by Hyperion.
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
					errors <- err
				}
			}
		}
	}()

	go func() {
		<-c.wait
		for {
			mtype, message, readErr := c.conn.Read(c.Ctx)
			if readErr != nil {
				errors <- readErr
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
				raw, ok := getRaw(m, c, errors)
				if !ok {
					return
				}
				sendResult(raw, results, errors)
			}(message)
		}
	}()

	return c, err
}

// getRaw parses out the message, and determines if it needs to be processed. It has been split out
// to facilitate unit tests.
func getRaw(m []byte, c *Client, errors chan error) (raw []interface{}, ok bool) {
	raw = make([]interface{}, 0)
	e := json.Unmarshal(m[2:], &raw)
	if e != nil {
		errors <- e
		return nil, false
	}

	// if it's not a string we're going to end up in trouble since reflection is involved, some socket.io
	// status messages pass arrays or objects here.
	switch raw[0].(type) {
	case string:
		break
	default:
		return nil, false
	}

	switch raw[0].(string) {
	case "lib_update":
		// track lib updates in stream.Client variables:
		update := raw[1].(map[string]interface{})
		if update["chain_id"] == nil || update["block_num"] == nil || update["block_id"] == nil {
			return nil, false
		}
		c.ChainId = update["chain_id"].(string)
		c.LibNum = uint32(math.Round(update["block_num"].(float64)))
		c.LibId = update["block_id"].(string)
		return nil, false
	case "message":
		break
	default:
		// everything else we don't care
		return nil, false
	}
	return raw, true
}

// sendResult performs final processing of the message, and forwards along if it is valid.
func sendResult(raw []interface{}, results chan HyperionResponse, errors chan error) {
	if len(raw) != 2 {
		return
	}
	var e error

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
	case "delta_trace":
		d := &DeltaTrace{}
		e = json.Unmarshal([]byte(raw[1].(map[string]interface{})["message"].(string)), d)
		if e != nil {
			errors <- e
			return
		}
		results <- d
	case "action_trace":
		a := &ActionTrace{}
		e = json.Unmarshal([]byte(raw[1].(map[string]interface{})["message"].(string)), a)
		if e != nil {
			errors <- e
			return
		}
		results <- a
	}
}

// StreamActions will emit an action stream request to Hyperion. Note that only one stream subscription is supported
// in this library to keep things simple.
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

// StreamDeltas will emit an delta stream request to Hyperion.
func (c *Client) StreamDeltas(req *DeltasReq) error {
	if c.subscribed {
		return BusyError{}
	}
	j, err := req.ToJson()
	if err != nil {
		return err
	}
	err = c.conn.Write(c.Ctx, websocket.MessageText, append(append([]byte(`420["delta_stream_request",`), j...), []byte("]")...))
	if err != nil {
		return err
	}
	c.subscribed = true
	close(c.wait)
	return nil
}

// ExitError is used when the socket.io-specific exit message is received
type ExitError struct{}

// Error satisfies the error interface
func (e ExitError) Error() string {
	return "websocket closed"
}

// BusyError is used when a socket already has a subscription
type BusyError struct{}

// Error satisfies the error interface
func (b BusyError) Error() string {
	return "websocket subscription already active, please use a new client for additional subscriptions"
}
