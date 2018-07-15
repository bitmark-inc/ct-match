package bitmarksdk

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/sha3"
)

const (
	assetTag                 = uint64(2)
	issueTag                 = uint64(3)
	transferUnratifiedTag    = uint64(4)
	transferCountersignedTag = uint64(5)

	minNameLength        = 1
	maxNameLength        = 64
	maxMetadataLength    = 2048
	minFingerprintLength = 1
	maxFingerprintLength = 1024
	assetIndexLength     = 64
	merkleDigestLength   = 32
)

// TODO: refine errors
var (
	ErrInvalidLength  = errors.New("invalid length")
	ErrInvalidAccount = errors.New("invalid account")
)

var nonceIndex uint64

type AssetRecord struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Metadata    string `json:"metadata"`
	Registrant  string `json:"registrant"`
	Signature   string `json:"signature"`
}

func NewAssetRecord(name, fingerprint string, metadata map[string]string, registrant *Account) (*AssetRecord, error) {
	parts := make([]string, 0, len(metadata)*2)
	for key, val := range metadata {
		if key == "" || val == "" {
			continue
		}
		parts = append(parts, key, val)
	}
	compactMetadata := strings.Join(parts, "\u0000")

	if utf8.RuneCountInString(name) < minNameLength || utf8.RuneCountInString(name) > maxNameLength {
		return nil, errors.New("property name not set or exceeds the maximum length (64 Unicode characters)")
	}

	if utf8.RuneCountInString(fingerprint) < minFingerprintLength || utf8.RuneCountInString(fingerprint) > maxFingerprintLength {
		return nil, errors.New("property fingerprint not set or exceeds the maximum length (1024 Unicode characters)")
	}

	if utf8.RuneCountInString(compactMetadata) > maxMetadataLength {
		return nil, errors.New("property metadata exceeds the maximum length (1024 Unicode characters)")
	}

	if registrant == nil {
		return nil, errors.New("registrant not set")
	}

	// pack and sign
	message := toVarint64(assetTag)
	message = appendString(message, name)
	message = appendString(message, fingerprint)
	message = appendString(message, compactMetadata)
	message = appendBytes(message, registrant.bytes())
	signature := hex.EncodeToString(registrant.AuthKey.Sign(message))

	return &AssetRecord{name, fingerprint, compactMetadata, registrant.AccountNumber(), signature}, nil
}

type IssueRecord struct {
	AssetIndex string `json:"asset"`
	Owner      string `json:"owner"`
	Nonce      uint64 `json:"nonce"`
	Signature  string `json:"signature"`
}

func NewIssueRecord(assetIndex string, issuer *Account) (*IssueRecord, error) {
	assetIndexBytes, err := hex.DecodeString(assetIndex)
	if err != nil || len(assetIndexBytes) != assetIndexLength {
		return nil, ErrInvalidLength
	}

	if issuer == nil {
		return nil, ErrInvalidAccount
	}

	atomic.AddUint64(&nonceIndex, 1)
	nonce := uint64(time.Now().UTC().Unix())*1000 + nonceIndex%1000

	// pack and sign
	message := toVarint64(issueTag)
	message = appendBytes(message, assetIndexBytes)
	message = appendBytes(message, issuer.bytes())
	message = appendUint64(message, nonce)
	signature := hex.EncodeToString(issuer.AuthKey.Sign(message))

	return &IssueRecord{
		assetIndex,
		issuer.AccountNumber(),
		nonce,
		signature,
	}, nil
}

func NewIssueRecordWithNonce(assetIndex string, issuer *Account, nonce uint64) (*IssueRecord, error) {
	assetIndexBytes, err := hex.DecodeString(assetIndex)
	if err != nil || len(assetIndexBytes) != assetIndexLength {
		return nil, ErrInvalidLength
	}

	if issuer == nil {
		return nil, ErrInvalidAccount
	}

	// pack and sign
	message := toVarint64(issueTag)
	message = appendBytes(message, assetIndexBytes)
	message = appendBytes(message, issuer.bytes())
	message = appendUint64(message, nonce)
	signature := hex.EncodeToString(issuer.AuthKey.Sign(message))

	return &IssueRecord{
		assetIndex,
		issuer.AccountNumber(),
		nonce,
		signature,
	}, nil
}

func (i *IssueRecord) Id() (string, error) {
	assetIndex, err := hex.DecodeString(i.AssetIndex)
	if err != nil || len(assetIndex) != assetIndexLength {
		return "", ErrInvalidLength
	}

	sig, err := hex.DecodeString(i.Signature)
	if err != nil {
		return "", ErrInvalidLength
	}

	// pack and sign
	message := toVarint64(issueTag)
	message = appendBytes(message, assetIndex)
	message = appendAccount(message, i.Owner)
	message = appendUint64(message, i.Nonce)
	message = appendBytes(message, sig)

	txIndex := sha3.Sum256(message)
	return hex.EncodeToString(txIndex[:]), nil
}

func NewIssueRecords(assetIndex string, issuer *Account, quantity int, nonces ...uint64) ([]*IssueRecord, error) {
	issues := make([]*IssueRecord, quantity)
	if nonces != nil {
		if len(nonces) != quantity {
			return nil, errors.New("the nonce count is not match the issue quantity")
		}
		for i, nonce := range nonces {
			var issue *IssueRecord

			issue, err := NewIssueRecordWithNonce(assetIndex, issuer, nonce)
			if err != nil {
				return nil, err
			}
			issues[i] = issue
		}
	} else {
		for i := 0; i < quantity; i++ {
			var issue *IssueRecord

			issue, err := NewIssueRecord(assetIndex, issuer)
			if err != nil {
				return nil, err
			}
			issues[i] = issue
		}
	}
	return issues, nil
}

type TransferRecord struct {
	Link      string `json:"link"`
	Owner     string `json:"owner"`
	Signature string `json:"signature"`
}

func NewTransferRecord(txId string, receiver string, owner *Account) (*TransferRecord, error) {
	link, err := hex.DecodeString(txId)
	if err != nil || len(link) != merkleDigestLength {
		return nil, ErrInvalidLength
	}

	if owner == nil {
		return nil, ErrInvalidAccount
	}

	// pack and sign
	message := toVarint64(transferUnratifiedTag)
	message = appendBytes(message, link)
	message = append(message, 0) // payment not supported
	message = appendAccount(message, receiver)
	signature := hex.EncodeToString(owner.AuthKey.Sign(message))

	return &TransferRecord{txId, receiver, signature}, nil
}

func (t TransferRecord) Id() (string, error) {
	link, err := hex.DecodeString(t.Link)
	if err != nil || len(link) != merkleDigestLength {
		return "", ErrInvalidLength
	}

	sig, err := hex.DecodeString(t.Signature)
	if err != nil {
		return "", ErrInvalidLength
	}

	// pack and sign
	message := toVarint64(transferUnratifiedTag)
	message = appendBytes(message, link)
	message = append(message, 0) // payment not supported
	message = appendAccount(message, t.Owner)
	message = appendBytes(message, sig)

	txIndex := sha3.Sum256(message)
	return hex.EncodeToString(txIndex[:]), nil
}

type TransferOfferRecord struct {
	Bitmark   *Bitmark `json:"bitmark,omitempty"`
	Link      string   `json:"link"`
	Owner     string   `json:"owner"`
	Signature string   `json:"signature"`
}

func (t TransferOfferRecord) String() string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}

type CountersignedTransferRecord struct {
	Link             string `json:"link"`
	Owner            string `json:"owner"`
	Signature        string `json:"signature"`
	Countersignature string `json:"countersignature,omitempty"`
}

func NewTransferOffer(bitmark *Bitmark, txId, receiver string, sender *Account) (*TransferOfferRecord, error) {
	link, err := hex.DecodeString(txId)
	if err != nil || len(link) != merkleDigestLength {
		return nil, ErrInvalidLength
	}

	if sender == nil {
		return nil, ErrInvalidAccount
	}

	// pack and sign
	message := toVarint64(transferCountersignedTag)
	message = appendBytes(message, link)
	message = append(message, 0) // payment not supported
	message = appendAccount(message, receiver)
	signature := hex.EncodeToString(sender.AuthKey.Sign(message))
	return &TransferOfferRecord{bitmark, txId, receiver, signature}, nil
}

func (t *TransferOfferRecord) Countersign(receiver *Account) (*CountersignedTransferRecord, error) {
	link, err := hex.DecodeString(t.Link)
	if err != nil || len(link) != merkleDigestLength {
		return nil, ErrInvalidLength
	}

	if receiver == nil || t.Owner != receiver.AccountNumber() {
		return nil, ErrInvalidAccount
	}

	sig, err := hex.DecodeString(t.Signature)
	if err != nil {
		return nil, ErrInvalidLength
	}

	// pack and sign
	message := toVarint64(transferCountersignedTag)
	message = appendBytes(message, link)
	message = append(message, 0) // payment not supported
	message = appendBytes(message, receiver.bytes())
	message = appendBytes(message, sig)

	return &CountersignedTransferRecord{t.Link, t.Owner, t.Signature, hex.EncodeToString(receiver.AuthKey.Sign(message))}, nil
}

func (ct *CountersignedTransferRecord) Id() (string, error) {
	link, err := hex.DecodeString(ct.Link)
	if err != nil || len(link) != merkleDigestLength {
		return "", ErrInvalidLength
	}

	sig, err := hex.DecodeString(ct.Signature)
	if err != nil {
		return "", ErrInvalidLength
	}

	countersig, err := hex.DecodeString(ct.Countersignature)
	if err != nil {
		return "", ErrInvalidLength
	}

	// pack and sign
	message := toVarint64(transferCountersignedTag)
	message = appendBytes(message, link)
	message = append(message, 0) // payment not supported
	message = appendAccount(message, ct.Owner)
	message = appendBytes(message, sig)
	message = appendBytes(message, countersig)

	txIndex := sha3.Sum256(message)
	return hex.EncodeToString(txIndex[:]), nil
}

const varint64MaximumBytes = 9

func appendString(buffer []byte, s string) []byte {
	l := toVarint64(uint64(len(s)))
	buffer = append(buffer, l...)
	return append(buffer, s...)
}

func appendAccount(buffer []byte, acctNo string) []byte {
	acctBytes := fromBase58(acctNo)
	return appendBytes(buffer, acctBytes[:len(acctBytes)-checksumLength])
}

func appendBytes(buffer []byte, data []byte) []byte {
	l := toVarint64(uint64(len(data)))
	buffer = append(buffer, l...)
	buffer = append(buffer, data...)
	return buffer
}

func appendUint64(buffer []byte, value uint64) []byte {
	valueBytes := toVarint64(value)
	buffer = append(buffer, valueBytes...)
	return buffer
}

func toVarint64(value uint64) []byte {
	result := make([]byte, 0, varint64MaximumBytes)
	if value < 0x80 {
		result = append(result, byte(value))
		return result
	}
	for i := 0; i < varint64MaximumBytes && value != 0; i++ {
		ext := uint64(0x80)
		if value < 0x80 {
			ext = 0x00
		}
		result = append(result, byte(value|ext))
		value >>= 7
	}
	return result
}

type Bitmark struct {
	HeadId      string       `json:"head_id"`
	Owner       string       `json:"owner"`
	AssetId     string       `json:"asset_id"`
	Id          string       `json:"id"`
	Issuer      string       `json:"issuer"`
	IssuedAt    time.Time    `json:"issued_at"`
	Head        string       `json:"head"`
	Status      string       `json:"status"`
	BlockNumber uint         `json:"block_number"`
	Offset      uint         `json:"offset"`
	CreatedAt   time.Time    `json:"created_at"`
	ConfirmedAt time.Time    `json:"confirmed_at"`
	Provenance  []Provenance `json:"provenance"`
	Asset       Asset        `json:"asset"`
}

type Provenance struct {
	TxId   string `json:"tx_id"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

type Asset struct {
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Fingerprint string            `json:"fingerprint"`
	Metadata    map[string]string `json:"metadata"`
	Registrant  string            `json:"registrant"`
	Status      string            `json:"status"`
	BlockNumber int               `json:"block_number"`
	BlockOffset int               `json:"block_offset"`
	ExpiresAt   string            `json:"expires_at"`
	Offset      int               `json:"offset"`
}
