package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmark-sdk-go/account"
	"github.com/bitmark-inc/bitmark-sdk-go/asset"
	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"
	"github.com/bitmark-inc/ct-match/util"
)

const (
	ProcessReceivingTrialBitmarkFromMatchingService int = iota
	ProcessReceivingTrialBitmarkFromSponsor         int = iota
)

type Participant struct {
	Account                  account.Account
	Name                     string
	conf                     ParticipantsConf
	Identities               map[string]string
	HoldingConsentBitmarkIDs []string
	IssuedMedicalData        map[string]string // Map between a consent tx and a bitmark id of medical data
}

func newParticipant(conf ParticipantsConf) (*Participant, error) {
	acc, err := account.New()
	if err != nil {
		return nil, err
	}

	return &Participant{
		Account:           acc,
		Name:              "Participant " + util.ShortenAccountNumber(acc.AccountNumber()),
		conf:              conf,
		IssuedMedicalData: make(map[string]string),
	}, nil
}

func (p *Participant) ProcessRecevingTrialBitmark(fromcase int) ([]string, error) {
	// p.print("Participant has " + strconv.Itoa(len(p.waitingTransferOffer)) + " transfer requests")
	bitmarkIDs := make([]string, 0)
	var prob float64
	switch fromcase {
	case ProcessReceivingTrialBitmarkFromMatchingService:
		prob = p.conf.AcceptTrialInviteProb
	case ProcessReceivingTrialBitmarkFromSponsor:
		prob = p.conf.AcceptMatchProb
	}

	builder := bitmark.NewQueryParamsBuilder().
		OfferTo(p.Account.AccountNumber()).
		LoadAsset(true)

	bitmarks, err := bitmark.List(builder)
	if err != nil {
		return nil, err
	}

	for _, b := range bitmarks {
		willAccept := util.RandWithProb(prob)

		if willAccept {
			offerResponseParam := bitmark.NewTransferResponseParams(b, bitmark.Accept)
			offerResponseParam.Sign(p.Account)
			if err := bitmark.Respond(offerResponseParam); err != nil {
				return nil, err
			}

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s accepted consent bitmark for %s from %s and is considering participation.\n", p.Name, b.Asset.Name, p.Identities[b.Offer.From])
				bitmarkIDs = append(bitmarkIDs, b.Id)
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s signed for acceptance of consent bitmark from %s and has been successfully entered as a participant in %s.\n", p.Name, p.Identities[b.Offer.From], b.Asset.Name)
			}
		} else {
			offerResponseParam := bitmark.NewTransferResponseParams(b, bitmark.Reject)
			offerResponseParam.Sign(p.Account)
			if err := bitmark.Respond(offerResponseParam); err != nil {
				return nil, err
			}

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s rejected consent bitmark for %s from %s.\n", p.Name, b.Asset.Name, p.Identities[b.Offer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s has opted to reject acceptance of consent bitmark from %s and refused the invitation to participate in %s.\n", p.Name, p.Identities[b.Offer.From], b.Asset.Name)
			}
		}
	}

	p.HoldingConsentBitmarkIDs = bitmarkIDs

	return bitmarkIDs, nil
}

func (p *Participant) SendBackTrialBitmark() error {
	for consentBitmarkID, medicalBitmarkID := range p.IssuedMedicalData {
		consentBitmarkInfo, err := bitmark.Get(consentBitmarkID, true)
		if err != nil {
			return err
		}

		matchingServiceAccountNumber := consentBitmarkInfo.Issuer

		// Transfer medical bitmark
		medicalOfferParam := bitmark.NewOfferParams(matchingServiceAccountNumber, nil)
		medicalOfferParam.FromBitmark(medicalBitmarkID)
		medicalOfferParam.Sign(p.Account)
		if err := bitmark.Offer(medicalOfferParam); err != nil {
			return err
		}

		// Also transfer the consent bitmark
		consentOfferParam := bitmark.NewOfferParams(matchingServiceAccountNumber, nil)
		consentOfferParam.FromBitmark(consentBitmarkID)
		consentOfferParam.Sign(p.Account)
		if err := bitmark.Offer(consentOfferParam); err != nil {
			return err
		}

		// Get bitmark info of trial
		medicalBitmarkInfo, err := bitmark.Get(medicalBitmarkID, true)
		if err != nil {
			return err
		}

		identityForReceiver := p.Identities[matchingServiceAccountNumber]

		fmt.Printf("%s issued health data bitmark for %s and sent it to %s for evaluation along with consent bitmark.\n", p.Name, medicalBitmarkInfo.Asset.Name, identityForReceiver)
	}
	return nil
}

func (p *Participant) IssueMedicalDataBitmark() ([]string, error) {
	medicalBitmarkIDs := make([]string, 0)
	for _, consentBitmarkID := range p.HoldingConsentBitmarkIDs {
		if !util.RandWithProb(p.conf.SubmitDataProb) {
			continue
		}

		consentBitmarkInfo, err := bitmark.Get(consentBitmarkID, true)
		if err != nil {
			return nil, err
		}

		medicalContent := "MEDICAL DATA\n" + util.RandStringBytesMaskImprSrc(1000)

		assetParam, err := asset.NewRegistrationParams(
			"health_data_"+p.Name+"_"+p.Identities[consentBitmarkInfo.Asset.Registrant],
			map[string]string{
				"Type":          "Health Data",
				"Trial Bitmark": consentBitmarkID,
			},
		)

		assetParam.SetFingerprint([]byte(medicalContent))
		assetParam.Sign(p.Account)
		assetID, err := asset.Register(assetParam)
		if err != nil {
			return nil, err
		}

		issueParam := bitmark.NewIssuanceParams(
			assetID,
			1,
		)
		issueParam.Sign(p.Account)
		bitmarkIDs, err := bitmark.Issue(issueParam)
		if err != nil {
			return nil, err
		}

		bitmarkID := bitmarkIDs[0]
		medicalBitmarkIDs = append(medicalBitmarkIDs, bitmarkID)

		p.IssuedMedicalData[consentBitmarkID] = bitmarkID
	}

	return medicalBitmarkIDs, nil
}

func (p *Participant) print(a ...interface{}) {
	fmt.Println("["+p.Name+"] ", a)
}
