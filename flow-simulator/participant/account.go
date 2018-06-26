package participant

import (
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

func (p *Participant) ProcessRecevingTrialBitmark(offerID string) (string, error) {
	c.Println(tag + "New trial opportunity from Sponsor 1. Do you want to accept it? (y/N)")
	isAccepted := util.AskForConfirmation()
	var action string
	if isAccepted {
		action = "accept"
	} else {
		action = "reject"
	}

	transferOffer, err := p.apiClient.GetTransferOfferById(offerID)
	if err != nil {
		return "", err
	}

	counterSign, err := transferOffer.Record.Countersign(p.Account)
	if err != nil {
		return "", err
	}

	return p.apiClient.CompleteTransferOffer(p.Account, offerID, action, counterSign.Countersignature)
}

func (p *Participant) SendBackTrialBitmark(bitmarkID, receiver string) (string, error) {
	return p.apiClient.Transfer(p.Account, bitmarkID, receiver)
}

func (p *Participant) AddTransferOffer(offerId string) {
	p.waitingTransferOffer = append(p.waitingTransferOffer, offerId)
}

func (p *Participant) print(a ...interface{}) {
	c.Println("["+p.Name+"] ", a)
}
