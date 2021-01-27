package stream

import "github.com/eoscanada/eos-go"

type HyperionReq interface {
	ToReq() ([]byte, error)
}

type DeltasReq struct {
	Code      eos.AccountName `json:"code"`
	Table     eos.Name        `json:"table"`
	Scope     eos.Name        `json:"scope"`
	Payer     eos.AccountName `json:"payer"`
	StartFrom interface{}     `json:"start_from"` // number or string
	ReadUntil interface{}     `json:"read_until"` // number or string
}

// TODO:
func NewDeltasReq() *DeltasReq {
	return nil
}

// TODO:
func (dr *DeltasReq) ToReq() ([]byte, error) {
	return nil, nil
}

type ReqFilter struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type ActionsReq struct {
	Contract  eos.AccountName `json:"contract"`
	Account   eos.AccountName `json:"account"`
	Action    eos.ActionName  `json:"action"`
	Filters   []*ReqFilter    `json:"filters"`
	StartFrom interface{}     `json:"start_from"`
	ReadUntil interface{}     `json:"read_until"`
}

//TODO
func NewActionsReq() *ActionsReq {
	return nil
}

//TODO
func (ar *ActionsReq) ToReq() ([]byte, error) {
	return nil, nil
}

type ResponseType string
type ResponseMode string

const (
	RespActionType ResponseType = "action"
	RespDeltaType  ResponseType = "delta"
	RespModeLive   ResponseMode = "live"
	RespModeHist   ResponseMode = "history"
)

type HyperionResponse interface {
	Type() ResponseType
	Mode() ResponseMode
	Action() (*Action, error)
	Delta() (*Delta, error)
}

type NotAction struct{}

func (na NotAction) Error() string {
	return "not an action"
}

type NotDelta struct{}

func (nd NotDelta) Error() string {
	return "not a delta"
}

type response struct {
	Type         ResponseType           `json:"type"`
	Mode         ResponseMode           `json:"mode"`
	Content      map[string]interface{} `json:"content"`
	Irreversible bool                   `json:"irreversible"`
}

// TODO
func (r *response) toAction() (*Action, error) {
	if r.Type == RespDeltaType {
		return nil, NotDelta{}
	}
	return nil, nil
}

// TODO
func (r *response) toDelta() (*Delta, error) {
	if r.Type == RespActionType {
		return nil, NotAction{}
	}
	return nil, nil
}

type Action struct {
	ESTimeStamp string `json:"@timestamp"`
	Notified    string `json:"notified"`
	mode        ResponseMode
	eos.TransactionReceipt
}

func (act *Action) Type() ResponseType {
	return RespActionType
}

func (act *Action) Mode() ResponseMode {
	return act.mode
}

func (act *Action) Action() (*Action, error) {
	return act, nil
}

func (act *Action) Delta() (*Delta, error) {
	return nil, NotDelta{}
}

type Delta struct {
	Code  eos.AccountName `json:"code"`
	Table eos.Name        `json:"table"`
	Scope eos.Name        `json:"scope"`
	Payer eos.AccountName `json:"payer"`
	Data  interface{}     `json:"data"` // likey map[string]interface{} or eos.HexData
	mode ResponseMode
}

func (d *Delta) Type() ResponseType {
	return RespDeltaType
}

func (d *Delta) Mode() ResponseMode {
	return d.mode
}

func (d *Delta) Action() (*Action, error) {
	return nil, NotDelta{}
}

func (d *Delta) Delta() (*Delta, error) {
	return d, nil
}
