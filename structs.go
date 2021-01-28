package stream

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"time"
)

type hyperionReq interface {
	toReq() ([]byte, error)
}

type DeltasReq struct {
	Code      eos.AccountName `json:"code"`
	Table     eos.Name        `json:"table"`
	Scope     eos.Name        `json:"scope"`
	Payer     eos.AccountName `json:"payer"`
	StartFrom interface{}     `json:"start_from"` // number or string
	ReadUntil interface{}     `json:"read_until"` // number or string
}

func NewDeltasReq(code string, table string, scope string, payer string) *DeltasReq {
	return ndr(code, table, scope, payer, nil, nil, 0, 0)
}

func NewDeltasReqByBlock(code string, table string, scope string, payer string, first int64, last int64) *DeltasReq {
	return ndr(code, table, scope, payer, nil, nil, first, last)
}

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

func NewActionsReq(contract string, account string, action string) *ActionsReq {
	return nar(contract, account, action, nil, nil, 0, 0)
}

func NewActionsReqByTime(contract string, account string, action string, start *time.Time, end *time.Time) *ActionsReq {
	return nar(contract, account, action, start, end, 0, 0)
}

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

func (ar *ActionsReq) AddFilter(f *ReqFilter) (ok bool) {
	if f == nil {
		return false
	}
	ar.Filters = append(ar.Filters, f)
	return true
}

func (ar *ActionsReq) ToJson() ([]byte, error) {
	return json.Marshal(ar)
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
	Action() (*ActionTrace, error)
	Delta() (*Delta, error)
}

type NotAction struct{}

func (NotAction) Error() string {
	return "not an action"
}

type NotDelta struct{}

func (NotDelta) Error() string {
	return "not a delta"
}

type UnknownType struct{}

func (UnknownType) Error() string {
	return "unknown response type"
}

type response struct {
	Type         ResponseType    `json:"type"`
	Mode         ResponseMode    `json:"mode"`
	Content      json.RawMessage `json:"content"`
	Irreversible bool            `json:"irreversible"`
}

func (r *response) Parse() (HyperionResponse, error) {
	switch r.Type {
	case RespActionType:
		a := &ActionTrace{}
		err := json.Unmarshal(r.Content, a)
		if err != nil {
			return nil, err
		}
		a.mode = r.Mode
		return a, nil
	case RespDeltaType:
		d := &Delta{}
		err := json.Unmarshal(r.Content, d)
		if err != nil {
			return nil, err
		}
		d.mode = r.Mode
		return d, nil
	default:
		return nil, UnknownType{}
	}
}

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

func (act *ActionTrace) Type() ResponseType {
	return RespActionType
}

func (act *ActionTrace) Mode() ResponseMode {
	return act.mode
}

func (act *ActionTrace) Action() (*ActionTrace, error) {
	return act, nil
}

func (act *ActionTrace) Delta() (*Delta, error) {
	return nil, NotDelta{}
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

type Delta struct {
	Code  eos.AccountName `json:"code"`
	Table eos.Name        `json:"table"`
	Scope eos.Name        `json:"scope"`
	Payer eos.AccountName `json:"payer"`
	Data  interface{}     `json:"data"` // likey map[string]interface{} or eos.HexData
	mode  ResponseMode
}

func (d *Delta) Type() ResponseType {
	return RespDeltaType
}

func (d *Delta) Mode() ResponseMode {
	return d.mode
}

func (d *Delta) Action() (*ActionTrace, error) {
	return nil, NotDelta{}
}

func (d *Delta) Delta() (*Delta, error) {
	return d, nil
}

func (d *Delta) ToJson() []byte {
	if d == nil {
		return nil
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil
	}
	return b
}

type LibUpdate struct {
	ChainId  string `json:"chain_id"`
	BlockNum uint32 `json:"block_num"`
	BlockId  string `json:"block_id"`
}
