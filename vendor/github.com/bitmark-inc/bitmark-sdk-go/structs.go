package bitmarksdk

import "time"

type transaction struct {
	TxId string `json:"txId"`
}

type assetAccessMeta struct {
	URL      string       `json:"url"`
	SessData *SessionData `json:"session_data"`
}

type accessByOwnership struct {
	assetAccessMeta
	Sender string `json:"sender"`
}

type AccessGrant struct {
	Id          string       `json:"id"`
	AssetId     string       `json:"asset_id"`
	SessionData *SessionData `json:"session_data,omitempty"`
	URL         string       `json:"url,omitempty"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	StartAt     time.Time    `json:"start_at"`
	EndAt       time.Time    `json:"end_at"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Duration struct {
	Years  uint64
	Months uint64
	Days   uint64
}
