package participant

import (
	"net/http"
	"strconv"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/util"
	"github.com/fatih/color"
)

var (
	c   = color.New(color.FgRed)
	tag = "[PARTICIPANT] "
)

type Participant struct {
	Account              *sdk.Account
	apiClient            *sdk.Client
	Name                 string
	conf                 config.ParticipantsConf
	waitingTransferOffer []string
	holdingConsentTxs    []string
}

func New(name string, client *sdk.Client, conf config.ParticipantsConf) (*Participant, error) {
	acc, err := client.CreateAccount()
	if err != nil {
		return nil, err
	}

	c.Println(tag + "Initialize participant with bitmark account: " + acc.AccountNumber())

	return &Participant{
		Account:   acc,
		apiClient: client,
		Name:      name,
		conf:      conf,
	}, nil
}

func (p *Participant) ProcessRecevingTrialBitmark() ([]string, error) {
	p.print("Participant has " + strconv.Itoa(len(p.waitingTransferOffer)) + " transfer requests")
	txIDs := make([]string, 0)
	for i, offerID := range p.waitingTransferOffer {
		isAccepted := util.RandWithProb(p.conf.AcceptTrialInviteProb)
		var action string
		if isAccepted {
			action = "accept"
		} else {
			action = "reject"
		}

		p.print("Process transfer " + strconv.Itoa(i) + " with action: " + action)

		transferOffer, err := p.apiClient.GetTransferOfferById(offerID)
		if err != nil {
			return nil, err
		}

		counterSign, err := transferOffer.Record.Countersign(p.Account)
		if err != nil {
			return nil, err
		}

		txID, err := p.apiClient.CompleteTransferOffer(p.Account, offerID, action, counterSign.Countersignature)
		if len(txID) > 0 {
			txIDs = append(txIDs, txID)
		}
	}

	p.holdingConsentTxs = txIDs

	return txIDs, nil
}

func (p *Participant) SendBackTrialBitmark(network string, httpClient *http.Client) ([]string, error) {
	for i, tx := range p.holdingConsentTxs {
		txInfo, err := util.GetTXInfo(tx, network, httpClient)
		if err != nil {
			return "", err
		}

		previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
		if err != nil {
			return "", err
		}

		p.apiClient.Transfer(p.Account, txInfo.BitmarkID, previousTxInfo.Owner)
	}
	return
}

func (p *Participant) IssueMedicalDataBitmark(name, consentBitmarkID string) (string, error) {
	medicalContent := "MEDICAL DATA\n" + util.RandStringBytesMaskImprSrc(1000)
	af := sdk.NewAssetFile("medical_data.txt", []byte(medicalContent), sdk.Private)
	bitmarkIDs, err := p.apiClient.IssueByAssetFile(p.Account, af, 1, &sdk.AssetInfo{
		Name: "Medical data for " + name,
		Metadata: map[string]string{
			"Consent_Bitmark": consentBitmarkID,
		},
	})

	if err != nil {
		return "", err
	}

	bitmarkID := bitmarkIDs[0]

	p.print("Issued medical data with bitmark id: ", bitmarkID)
	return bitmarkID, nil
}

func (p *Participant) AddTransferOffer(offerId string) {
	p.waitingTransferOffer = append(p.waitingTransferOffer, offerId)
}

func (p *Participant) print(a ...interface{}) {
	c.Println("["+p.Name+"] ", a)
}
