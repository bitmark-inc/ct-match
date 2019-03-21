package bitmarksdk

import (
	"errors"
	"fmt"
	"net/http"
)

type Config struct {
	HTTPClient *http.Client
	Network    string

	APIEndpoint string
	KeyEndpoint string
}

type Client struct {
	Network Network
	service *Service
}

func NewClient(cfg *Config) *Client {
	var apiEndpoint string
	var keyEndpoint string
	var network Network
	switch cfg.Network {
	case "testnet":
		apiEndpoint = "https://api.test.bitmark.com"
		keyEndpoint = "https://key.test.bitmarkaccountassets.com"
		network = Testnet
	case "livenet":
		apiEndpoint = "https://api.bitmark.com"
		keyEndpoint = "https://key.bitmarkaccountassets.com"
		network = Livenet
	default:
		panic("unsupported network")
	}

	// allow endpoints customization
	if cfg.APIEndpoint != "" {
		apiEndpoint = cfg.APIEndpoint
	}
	if cfg.KeyEndpoint != "" {
		keyEndpoint = cfg.KeyEndpoint
	}

	svc := &Service{cfg.HTTPClient, apiEndpoint, keyEndpoint}
	return &Client{network, svc}
}

func (c *Client) CreateAccount() (*Account, error) {
	seed, err := NewSeed(SeedVersion1, c.Network)
	if err != nil {
		return nil, err
	}

	authKey, err := NewAuthKey(seed)
	if err != nil {
		return nil, err
	}

	encrKey, err := NewEncrKey(seed)
	if err != nil {
		return nil, err
	}

	account := &Account{seed: seed, AuthKey: authKey, EncrKey: encrKey}

	if err := c.service.registerEncPubkey(account); err != nil {
		return nil, err
	}
	return account, nil
}

func (c *Client) RestoreAccountFromSeed(s string) (*Account, error) {
	seed, err := SeedFromBase58(s)
	if err != nil {
		return nil, err
	}

	if seed.network != c.Network {
		return nil, fmt.Errorf("trying to restore %s account in %s environment", seed.network, c.Network)
	}

	authKey, err := NewAuthKey(seed)
	if err != nil {
		return nil, err
	}

	encrKey, err := NewEncrKey(seed)
	if err != nil {
		return nil, err
	}

	return &Account{seed: seed, AuthKey: authKey, EncrKey: encrKey}, nil
}

func (c *Client) IssueByAssetFile(acct *Account, af *AssetFile, quantity int, info *AssetInfo) ([]string, error) {
	var asset *AssetRecord

	if info != nil {
		var err error
		asset, err = NewAssetRecord(info.Name, af.Fingerprint, info.Metadata, acct)
		if err != nil {
			return nil, err
		}
	}

	issues, err := NewIssueRecords(af.Id(), acct, quantity)
	if err != nil {
		return nil, err
	}

	if uerr := c.service.uploadAsset(acct, af); uerr != nil {
		return nil, uerr
	}
	bitmarkIds, err := c.service.createIssueTx(asset, issues)
	return bitmarkIds, err
}

func (c *Client) IssueByAssetFileWithNonces(acct *Account, af *AssetFile, info *AssetInfo, nonces []uint64) ([]string, error) {
	var asset *AssetRecord

	if info != nil {
		var err error
		asset, err = NewAssetRecord(info.Name, af.Fingerprint, info.Metadata, acct)
		if err != nil {
			return nil, err
		}
	}

	issues, err := NewIssueRecords(af.Id(), acct, len(nonces), nonces...)
	if err != nil {
		return nil, err
	}

	if uerr := c.service.uploadAsset(acct, af); uerr != nil {
		return nil, uerr
	}
	bitmarkIds, err := c.service.createIssueTx(asset, issues)
	return bitmarkIds, err
}

func (c *Client) IssueByAssetId(acct *Account, assetId string, quantity int) ([]string, error) {
	issues, err := NewIssueRecords(assetId, acct, quantity)
	if err != nil {
		return nil, err
	}

	bitmarkIds, err := c.service.createIssueTx(nil, issues)
	return bitmarkIds, err
}

func (c *Client) Issue(asset *AssetRecord, issues []*IssueRecord) ([]string, error) {
	bitmarkIds, err := c.service.createIssueTx(asset, issues)
	return bitmarkIds, err
}

func (c *Client) Transfer(acct *Account, bitmarkId, receiver string) (string, error) {
	// access, aerr := c.service.getAssetAccess(acct, bitmarkId)
	// if aerr != nil {
	// 	return "", aerr
	// }

	// if access.SessData != nil {
	// 	senderPublicKey, err := c.service.getEncPubkey(access.Sender)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	dataKey, err := dataKeyFromSessionData(acct, access.SessData, senderPublicKey)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	recipientEncrPubkey, err := c.service.getEncPubkey(receiver)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	data, err := createSessionData(acct, dataKey, recipientEncrPubkey)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	err = c.service.addSessionData(acct, bitmarkId, receiver, data)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	bmk, err := c.service.getBitmark(bitmarkId)
	if err != nil {
		return "", err
	}

	if acct.AccountNumber() != bmk.Owner {
		return "", errors.New("not bitmark owner")
	}

	tr, err := NewTransferRecord(bmk.HeadId, receiver, acct)
	if err != nil {
		return "", err
	}

	return c.service.createTransferTx(tr)
}

func (c *Client) SignTransferOffer(sender *Account, bitmarkId, receiver string, includeBitmark bool) (*TransferOfferRecord, error) {
	// access, aerr := c.service.getAssetAccess(sender, bitmarkId)
	// if aerr != nil {
	// 	return nil, aerr
	// }

	// if access.SessData != nil {
	// 	senderPublicKey, err := c.service.getEncPubkey(access.Sender)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	dataKey, err := dataKeyFromSessionData(sender, access.SessData, senderPublicKey)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	recipientEncrPubkey, err := c.service.getEncPubkey(receiver)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	data, err := createSessionData(sender, dataKey, recipientEncrPubkey)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	err = c.service.addSessionData(sender, bitmarkId, receiver, data)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	bmk, err := c.service.getBitmark(bitmarkId)
	if err != nil {
		return nil, err
	}

	if sender.AccountNumber() != bmk.Owner {
		return nil, errors.New("not bitmark owner")
	}

	if includeBitmark {
		return NewTransferOffer(bmk, bmk.HeadId, receiver, sender)
	}
	return NewTransferOffer(nil, bmk.HeadId, receiver, sender)
}

func (c *Client) SubmitTransferOffer(sender *Account, t *TransferOfferRecord, extraInfo interface{}) (string, error) {
	return c.service.submitTransferOffer(sender, t, extraInfo)
}

func (c *Client) GetTransferOfferByAccount(acct *Account) (map[string][]*TransferOffer, error) {
	return c.service.getTransferOfferByAccount(acct)
}

func (c *Client) GetTransferOfferById(offerId string) (*TransferOffer, error) {
	return c.service.getTransferOfferById(offerId)
}

func (c *Client) CompleteTransferOffer(receiver *Account, offerId, action, countersignature string) (string, error) {
	return c.service.completeTransferOffer(receiver, offerId, action, countersignature)
}

func (c *Client) CountersignedTransfer(t *CountersignedTransferRecord) (string, error) {
	return c.service.createCountersignTransferTx(t)
}

func (c *Client) CountersignTransfer(receiver *Account, t *TransferOfferRecord) (string, error) {
	record, err := t.Countersign(receiver)
	if err != nil {
		return "", err
	}
	return c.service.createCountersignTransferTx(record)
}

func (c *Client) DownloadAsset(acct *Account, bitmarkId string) (string, []byte, error) {
	access, err := c.service.getAssetAccess(acct, bitmarkId)
	if err != nil {
		return "", nil, err
	}

	fileName, content, err := c.service.getAssetContent(access.URL)
	if err != nil {
		return "", nil, err
	}
	if fileName == "" {
		fileName = bitmarkId
	}

	if access.SessData == nil { // public asset
		return fileName, content, nil
	}

	encrPubkey, err := c.service.getEncPubkey(access.Sender)
	if err != nil {
		return "", nil, fmt.Errorf("fail to get enc public key: %s", err.Error())
	}

	dataKey, err := dataKeyFromSessionData(acct, access.SessData, encrPubkey)
	if err != nil {
		return "", nil, err
	}

	plaintext, err := dataKey.Decrypt(content)
	if err != nil {
		return "", nil, err
	}

	return fileName, plaintext, nil
}

func (c *Client) GrantAssetAccess(acct *Account, bitmarkId, to string, startAt int64, duration Duration) (*AccessGrant, error) {
	access, err := c.service.getAssetAccess(acct, bitmarkId)
	if err != nil {
		return nil, err
	}
	if access.SessData == nil {
		return nil, errors.New("no need to grant access on public asset")
	}

	dataKey, err := dataKeyFromSessionData(acct, access.SessData, acct.EncrKey.PublicKeyBytes())
	if err != nil {
		return nil, err
	}

	recipientEncrPubkey, err := c.service.getEncPubkey(to)
	if err != nil {
		return nil, err
	}

	data, err := createSessionData(acct, dataKey, recipientEncrPubkey)
	if err != nil {
		return nil, err
	}

	return c.service.createAccessGrant(acct, bitmarkId, to, data, startAt, duration)
}

func (c *Client) RevokeAssetAccess(acct *Account, grantId string) error {
	return c.service.deleteAccessGrant(acct, grantId)
}

func (c *Client) ListGrantedAssetAccess(accountNumber, direction string) ([]*AccessGrant, error) {
	return c.service.queryAccessGrant(accountNumber, direction)
}

func (c *Client) DownloadAssetByGrant(acct *Account, grantId string) ([]byte, error) {
	access, err := c.service.getAccessGrant(acct, grantId)
	if err != nil {
		return nil, err
	}

	_, content, err := c.service.getAssetContent(access.URL)
	if err != nil {
		return nil, err
	}

	encrPubkey, err := c.service.getEncPubkey(access.From)
	if err != nil {
		return nil, fmt.Errorf("fail to get enc public key: %s", err.Error())
	}

	dataKey, err := dataKeyFromSessionData(acct, access.SessionData, encrPubkey)
	if err != nil {
		return nil, err
	}

	plaintext, err := dataKey.Decrypt(content)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (c *Client) QueryBitmarks(filter *BitmarkFilter) ([]*Bitmark, error) {
	return c.service.queryBitmarks(filter)
}
