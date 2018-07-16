package participant

import (
	"fmt"
	"net/http"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/util"
	"github.com/fatih/color"
)

var (
	c   = color.New(color.FgRed)
	tag = "[PARTICIPANT] "
)

const (
	ProcessReceivingTrialBitmarkFromMatchingService int = iota
	ProcessReceivingTrialBitmarkFromSponsor         int = iota
)

type Participant struct {
	Account              *sdk.Account
	apiClient            *sdk.Client
	Name                 string
	conf                 config.ParticipantsConf
	Identities           map[string]string
	waitingTransferOffer []string
	HoldingConsentTxs    []string
	issuedMedicalData    map[string]string // Map between a consent tx and a bitmark id of medical data
}

func New(client *sdk.Client, conf config.ParticipantsConf) (*Participant, error) {
	acc, err := client.CreateAccount()
	if err != nil {
		return nil, err
	}

	// c.Println(tag + "Initialize participant with bitmark account: " + acc.AccountNumber())

	return &Participant{
		Account:              acc,
		apiClient:            client,
		Name:                 "Participant " + util.ShortenAccountNumber(acc.AccountNumber()),
		conf:                 conf,
		waitingTransferOffer: make([]string, 0),
		issuedMedicalData:    make(map[string]string),
	}, nil
}

func (p *Participant) ProcessRecevingTrialBitmark(fromcase int, network string, httpClient *http.Client) ([]string, error) {
	// p.print("Participant has " + strconv.Itoa(len(p.waitingTransferOffer)) + " transfer requests")
	txIDs := make([]string, 0)
	var prob float64
	switch fromcase {
	case ProcessReceivingTrialBitmarkFromMatchingService:
		prob = p.conf.AcceptTrialInviteProb
	case ProcessReceivingTrialBitmarkFromSponsor:
		prob = p.conf.AcceptMatchProb
	}
	for _, offerID := range p.waitingTransferOffer {
		isAccepted := util.RandWithProb(prob)

		transferOffer, err := p.apiClient.GetTransferOfferById(offerID)
		if err != nil {
			return nil, err
		}

		bitmarkInfo, err := util.GetBitmarkInfo(transferOffer.BitmarkId, network, httpClient)
		if err != nil {
			return nil, err
		}

		var action string
		if isAccepted {
			action = "accept"

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s accepted consent bitmark for %s from %s and is considering participation.\n", p.Name, bitmarkInfo.Asset.Name, p.Identities[transferOffer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s signed for acceptance of consent bitmark from %s and has been successfully entered as a participant in %s.\n", p.Name, p.Identities[transferOffer.From], bitmarkInfo.Asset.Name)
			}
		} else {
			action = "reject"

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s rejected consent bitmark for %s from %s.\n", p.Name, bitmarkInfo.Asset.Name, p.Identities[transferOffer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s has opted to reject acceptance of consent bitmark from %s and refused the invitation to participate in %s.\n", p.Name, p.Identities[transferOffer.From], bitmarkInfo.Asset.Name)
			}
		}

		txID, err := util.TryToActionTransfer(transferOffer, action, p.Account, p.apiClient)
		if err != nil {
			return nil, err
		}

		if len(txID) > 0 {
			txIDs = append(txIDs, txID)
		}
	}

	p.HoldingConsentTxs = txIDs
	p.waitingTransferOffer = make([]string, 0) // Wipe out the waiting list

	return txIDs, nil
}

func (p *Participant) SendBackTrialBitmark(network string, httpClient *http.Client) (map[string]string, error) {
	transferOfferIDs := make(map[string]string)

	for tx, medicalBitmarkID := range p.issuedMedicalData {
		txInfo, err := util.GetTXInfo(tx, network, httpClient)
		if err != nil {
			return nil, err
		}

		if txInfo.Owner != p.Account.AccountNumber() {
			continue
		}

		previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
		if err != nil {
			return nil, err
		}

		trialOfferID, err := util.TryToSubmitTransfer(txInfo.BitmarkID, previousTxInfo.Owner, p.Account, p.apiClient)
		if err != nil {
			return nil, err
		}

		// Transfer also the medical data
		medicalOfferID, err := util.TryToSubmitTransfer(medicalBitmarkID, previousTxInfo.Owner, p.Account, p.apiClient)
		if err != nil {
			return nil, err
		}

		transferOfferIDs[trialOfferID] = medicalOfferID

		// Get bitmark info of trial
		bitmarkInfo, err := util.GetBitmarkInfo(txInfo.BitmarkID, network, httpClient)
		if err != nil {
			return nil, err
		}

		identityForReceiver := p.Identities[previousTxInfo.Owner]

		fmt.Printf("%s issued health data bitmark for %s and sent it to %s for evaluation along with consent bitmark.\n", p.Name, bitmarkInfo.Asset.Name, identityForReceiver)
	}
	return transferOfferIDs, nil
}

func (p *Participant) IssueMedicalDataBitmark(network string, httpClient *http.Client) ([]string, error) {
	medicalBitmarkIDs := make([]string, 0)
	for _, tx := range p.HoldingConsentTxs {
		if !util.RandWithProb(p.conf.SubmitDataProb) {
			continue
		}

		txInfo, err := util.GetTXInfo(tx, network, httpClient)
		if err != nil {
			return nil, err
		}

		bitmarkInfo, err := util.GetBitmarkInfo(txInfo.BitmarkID, network, httpClient)
		if err != nil {
			return nil, err
		}

		medicalContent := "MEDICAL DATA\n" + util.RandStringBytesMaskImprSrc(1000)
		af := sdk.NewAssetFile("medical_data.txt", []byte(medicalContent), sdk.Private)
		bitmarkIDs, err := p.apiClient.IssueByAssetFile(p.Account, af, 1, &sdk.AssetInfo{
			Name: "health_data_" + p.Name + "_" + p.Identities[bitmarkInfo.Asset.Registrant],
		})

		if err != nil {
			return nil, err
		}

		bitmarkID := bitmarkIDs[0]
		medicalBitmarkIDs = append(medicalBitmarkIDs, bitmarkID)

		p.issuedMedicalData[tx] = bitmarkID
	}

	return medicalBitmarkIDs, nil
}

func (p *Participant) AddTransferOffer(offerId string) {
	p.waitingTransferOffer = append(p.waitingTransferOffer, offerId)
}

func (p *Participant) print(a ...interface{}) {
	c.Println("["+p.Name+"] ", a)
}
