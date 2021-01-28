package stream

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"time"
)

type hyperionReq interface {
	toReq() ([]byte, error)
}

// DeltasReq is the query sent to Hyperion requesting a stream of table updates.
type DeltasReq struct {
	Code      eos.AccountName `json:"code"`
	Table     eos.Name        `json:"table"`
	Scope     eos.Name        `json:"scope"`
	Payer     eos.AccountName `json:"payer"`
	StartFrom interface{}     `json:"start_from"` // number or string
	ReadUntil interface{}     `json:"read_until"` // number or string
}

// NewDeltasReq is a request for table updates starting at the current head block.
func NewDeltasReq(code string, table string, scope string, payer string) *DeltasReq {
	return ndr(code, table, scope, payer, nil, nil, 0, 0)
}

// NewDeltasReqByBlock is a request for table updates with a specific block range. If last == 0 Hyperion will continue
// streaming data once it has caught up to the head block.
func NewDeltasReqByBlock(code string, table string, scope string, payer string, first int64, last int64) *DeltasReq {
	return ndr(code, table, scope, payer, nil, nil, first, last)
}

// NewDeltasReqByTime is a request for table updates with a specific time range. Note that it uses pointers, and passing
// in a nil pointer for the end time will instruct Hyperion to continue streaming once it has caught up to the current
// head block.
func NewDeltasReqByTime(code string, table string, scope string, payer string, start *time.Time, end *time.Time) *DeltasReq {
	return ndr(code, table, scope, payer, start, end, 0, 0)
}

func ndr(code string, table string, scope string, payer string, start *time.Time, end *time.Time, first int64, last int64) *DeltasReq {
	if scope == "" {
		scope = code
	}
	d := &DeltasReq{
		Code:  eos.AccountName(code),
		Table: eos.Name(table),
		Scope: eos.Name(scope),
		Payer: eos.AccountName(payer),
	}
	switch true {
	case start == nil && end == nil && first == 0 && last == 0:
		d.StartFrom, d.ReadUntil = 0, 0
	case start != nil:
		d.StartFrom = start.UTC().Format(time.RFC3339)
		fallthrough
	case end != nil:
		d.ReadUntil = end.UTC().Format(time.RFC3339)
	case first > 0:
		d.StartFrom = first
		fallthrough
	case last > 0 || last == -1:
		d.ReadUntil = last
	}
	return d
}

func (dr *DeltasReq) ToJson() ([]byte, error) {
	return json.Marshal(dr)
}

// ReqFilter instructs Hyperion to perform further filtering, more information at:
//  https://github.com/eosrio/hyperion-stream-client/tree/master#211-act-data-filters
// adding filters to an ActionsReq should be performed using the ActionsReq.AddFilter function.
type ReqFilter struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// ActionsReq is the query sent to Hyperion requesting it to stream action traces.
type ActionsReq struct {
	Contract  eos.AccountName `json:"contract"`
	Account   eos.AccountName `json:"account"`
	Action    eos.ActionName  `json:"action"`
	Filters   []*ReqFilter    `json:"filters"`
	StartFrom interface{}     `json:"start_from"`
	ReadUntil interface{}     `json:"read_until"`
}

// NewActionsReq is a request for action traces starting at the current head block.
func NewActionsReq(contract string, account string, action string) *ActionsReq {
	return nar(contract, account, action, nil, nil, 0, 0)
}

// NewActionsReqByTime is a request for action traces with a specific block range. If last == 0 Hyperion will continue
// streaming data once it has caught up to the head block.
func NewActionsReqByTime(contract string, account string, action string, start *time.Time, end *time.Time) *ActionsReq {
	return nar(contract, account, action, start, end, 0, 0)
}

// NewActionsReqByBlock is a request for action traces with a specific time range. Note that it uses pointers, and passing
// in a nil pointer for the end time will instruct Hyperion to continue streaming once it has caught up to the current
// head block.
func NewActionsReqByBlock(contract string, account string, action string, first int64, last int64) *ActionsReq {
	return nar(contract, account, action, nil, nil, first, last)
}

func nar(contract string, account string, action string, start *time.Time, end *time.Time, first int64, last int64) *ActionsReq {
	a := &ActionsReq{
		Contract: eos.AccountName(contract),
		Account:  eos.AccountName(account),
		Action:   eos.ActionName(action),
	}
	switch true {
	case start == nil && end == nil && first == 0 && last == 0:
		a.StartFrom, a.ReadUntil = 0, 0
	case start != nil:
		a.StartFrom = start.UTC().Format(time.RFC3339)
		fallthrough
	case end != nil:
		a.ReadUntil = end.UTC().Format(time.RFC3339)
	case first > 0:
		a.StartFrom = first
		fallthrough
	case last > 0 || last == -1:
		a.ReadUntil = last
	}
	return a
}

// AddFilter assists in appending a ReqFilter to the request.
func (ar *ActionsReq) AddFilter(f *ReqFilter) (ok bool) {
	if f == nil {
		return false
	}
	if ar.Filters == nil {
		ar.Filters = make([]*ReqFilter, 0)
	}
	ar.Filters = append(ar.Filters, f)
	return true
}

func (ar *ActionsReq) ToJson() ([]byte, error) {
	return json.Marshal(ar)
}

// ResponseType indicates if the HyperionResponse is an action trace or delta trace
type ResponseType string

// ResponseMode represents whether the data being streamed is live or historical
type ResponseMode string

const (
	RespActionType ResponseType = "action"
	RespDeltaType  ResponseType = "delta"
	RespModeLive   ResponseMode = "live"
	RespModeHist   ResponseMode = "history"
)

// HyperionResponse is the data being streamed over the results channel of the stream.Client it can be one of
// (at current) two types.
type HyperionResponse interface {
	Type() ResponseType
	Mode() ResponseMode
	Action() (*ActionTrace, error)
	Delta() (*DeltaTrace, error)
}

// ActionTrace holds a trace response, it differs somewhat for standard EOSIO structures. Note that the
// ActionTrace.Act.Data field is a map[string]interface that will mirror the raw JSON sent by Hyperion.
type ActionTrace struct {
	ActionOrdinal        uint32            `json:"action_ordinal"`
	CreatorActionOrdinal uint32            `json:"creator_action_ordinal"`
	ContextFree          bool              `json:"context_free"`
	Elapsed              string            `json:"elapsed"`
	TS                   string            `json:"@timestamp"`
	BlockNum             uint32            `json:"block_num"`
	Producer             eos.AccountName   `json:"producer"`
	TrxId                eos.HexBytes      `json:"trx_id"`
	GlobalSequence       uint64            `json:"global_sequence"`
	CodeSequence         uint32            `json:"code_sequence"`
	AbiSequence          uint32            `json:"abi_sequence"`
	Notified             []eos.AccountName `json:"notified"`

	Act struct {
		Account       eos.AccountName        `json:"account"`
		Name          eos.ActionName         `json:"name"`
		Authorization []eos.PermissionLevel  `json:"authorization"`
		Data          map[string]interface{} `json:"data"`
	} `json:"act"`

	Receipts []struct {
		Receiver       eos.AccountName       `json:"receiver"`
		GlobalSequence string                `json:"global_sequence"`
		RecvSequence   string                `json:"recv_sequence"`
		AuthSequence   []eos.PermissionLevel `json:"auth_sequence"`
	} `json:"receipts"`

	mode ResponseMode
}

// Type satisfies the HyperionResponse interface and will return what type of trace this is.
func (act *ActionTrace) Type() ResponseType {
	return RespActionType
}

// Mode satisfies the HyperionResponse interface and will return whether streaming live or historical events
func (act *ActionTrace) Mode() ResponseMode {
	return act.mode
}

// Action satisfies the HyperionResponse interface and will return a stream.ActionTrace if this is an action, otherwise
// it will return an error
func (act *ActionTrace) Action() (*ActionTrace, error) {
	return act, nil
}

// Delta satisfies the HyperionResponse interface and will return a stream.DeltaTrace if this is a delta, otherwise
// it will return an error
func (act *ActionTrace) Delta() (*DeltaTrace, error) {
	return nil, NotDeltaError{}
}

func (act *ActionTrace) ToJson() []byte {
	if act == nil {
		return nil
	}
	b, err := json.MarshalIndent(act, "", "  ")
	if err != nil {
		return nil
	}
	return b
}

type DeltaTrace struct {
	Code       eos.AccountName `json:"code"`
	Scope      eos.Name        `json:"scope"`
	Table      eos.Name        `json:"table"`
	PrimaryKey string          `json:"primary_key"`
	Payer      eos.AccountName `json:"payer"`
	TS         string          `json:"@timestamp"`
	Present    bool            `json:"present"`
	BlockNum   uint32          `json:"block_num"`
	BlockId    eos.HexBytes    `json:"block_id"`
	Data       interface{}     `json:"data"` // most likely map[string]interface{} or string
	mode       ResponseMode
}

// Type satisfies the HyperionResponse interface and will return what type of trace this is.
func (d *DeltaTrace) Type() ResponseType {
	return RespDeltaType
}

// Mode satisfies the HyperionResponse interface and will return whether streaming live or historical events
func (d *DeltaTrace) Mode() ResponseMode {
	return d.mode
}

// Action satisfies the HyperionResponse interface and will return a stream.ActionTrace if this is an action, otherwise
// it will return an error
func (d *DeltaTrace) Action() (*ActionTrace, error) {
	return nil, NotActionError{}
}

// Delta satisfies the HyperionResponse interface and will return a stream.DeltaTrace if this is a delta, otherwise
// it will return an error
func (d *DeltaTrace) Delta() (*DeltaTrace, error) {
	return d, nil
}

func (d *DeltaTrace) ToJson() []byte {
	if d == nil {
		return nil
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil
	}
	return b
}

type NotActionError struct{}

func (NotActionError) Error() string {
	return "not an action"
}

type NotDeltaError struct{}

func (NotDeltaError) Error() string {
	return "not a delta"
}

type UnknownTypeError struct{}

func (UnknownTypeError) Error() string {
	return "unknown response type"
}