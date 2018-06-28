package participant

import (
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

func New(name string, client *sdk.Client, conf config.ParticipantsConf) (*Participant, error) {
	acc, err := client.CreateAccount()
	if err != nil {
		return nil, err
	}

	// c.Println(tag + "Initialize participant with bitmark account: " + acc.AccountNumber())

	return &Participant{
		Account:              acc,
		apiClient:            client,
		Name:                 name,
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
				c.Printf("%s accepted consent bitmark %s for trial %s from %s and is considering participation.\n", p.Name, transferOffer.BitmarkId, bitmarkInfo.Asset.Name, p.Identities[transferOffer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				c.Printf("%s signed for acceptance of consent bitmark %s from %s and has been successfully entered as a participant in trial %s.\n", p.Name, transferOffer.BitmarkId, p.Identities[transferOffer.From], bitmarkInfo.Asset.Name)
			}
		} else {
			action = "reject"

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				c.Printf("%s rejected consent bitmark %s for trial %s from %s.\n", p.Name, transferOffer.BitmarkId, bitmarkInfo.Asset.Name, p.Identities[transferOffer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				c.Printf("%s has opted to reject acceptace of consent bitmark %s from %s and refused the invitation to participate in trial %s.\n", p.Name, transferOffer.BitmarkId, p.Identities[transferOffer.From], bitmarkInfo.Asset.Name)
			}
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

	p.HoldingConsentTxs = txIDs
	p.waitingTransferOffer = make([]string, 0) // Wipe out the waiting list

	return txIDs, nil
}

func (p *Participant) SendBackTrialBitmark(network string, httpClient *http.Client) (map[string]string, error) {
	transferOfferIDs := make(map[string]string)

	for _, tx := range p.HoldingConsentTxs {
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

		trialTransferOffer, err := sdk.NewTransferOffer(nil, tx, previousTxInfo.Owner, p.Account)
		if err != nil {
			return nil, err
		}

		trialOfferID, err := p.apiClient.SubmitTransferOffer(p.Account, trialTransferOffer, nil)
		if err != nil {
			return nil, err
		}

		// Transfer also the medical data
		medicalBitmarkID := p.issuedMedicalData[tx]
		medicalTransferOffer, err := sdk.NewTransferOffer(nil, medicalBitmarkID, previousTxInfo.Owner, p.Account)
		if err != nil {
			return nil, err
		}

		medicalOfferID, err := p.apiClient.SubmitTransferOffer(p.Account, medicalTransferOffer, nil)
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

		c.Printf("%s issued health data bitmark %s for trial %s and sent it to %s for evaluation along with consent bitmark %s.\n", p.Name, medicalBitmarkID, bitmarkInfo.Asset.Name, identityForReceiver, txInfo.BitmarkID)
	}
	return transferOfferIDs, nil
}

func (p *Participant) IssueMedicalDataBitmark(network string, httpClient *http.Client) ([]string, error) {
	bitmarkIDs := make([]string, 0)
	for _, tx := range p.HoldingConsentTxs {
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

		p.issuedMedicalData[tx] = bitmarkID
	}

	return bitmarkIDs, nil
}

func (p *Participant) AddTransferOffer(offerId string) {
	p.waitingTransferOffer = append(p.waitingTransferOffer, offerId)
}

func (p *Participant) print(a ...interface{}) {
	c.Println("["+p.Name+"] ", a)
}
