package bitmarksdk

import (
	"encoding/json"
	"time"
)

type TransferOffer struct {
	Id        string              `json:"id"`
	BitmarkId string              `json:"bitmark_id"`
	From      string              `json:"from"`
	To        string              `json:"to"`
	Status    string              `json:"status"`
	Record    TransferOfferRecord `json:"record"`
	Metadata  json.RawMessage     `json:"metadata"`
	CreatedAt time.Time           `json:"created_at"`
	TxId      string              `json:"txId"`
	Open      bool                `json:"open"`
}
