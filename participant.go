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

	bitmarks, assets, err := bitmark.List(builder)
	if err != nil {
		return nil, err
	}
	referencedAssets := make(map[string]*asset.Asset)
	for _, asset := range assets {
		referencedAssets[asset.ID] = asset
	}

	for _, b := range bitmarks {
		willAccept := util.RandWithProb(prob)

		if willAccept {
			offerResponseParam := bitmark.NewTransferResponseParams(b, bitmark.Accept)
			offerResponseParam.Sign(p.Account)
			if _, err := bitmark.Respond(offerResponseParam); err != nil {
				return nil, err
			}

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s accepted consent bitmark for %s from %s and is considering participation.\n", p.Name, referencedAssets[b.AssetID].Name, p.Identities[b.Offer.From])
				bitmarkIDs = append(bitmarkIDs, b.ID)
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s signed for acceptance of consent bitmark from %s and has been successfully entered as a participant in %s.\n", p.Name, p.Identities[b.Offer.From], referencedAssets[b.AssetID].Name)
			}
		} else {
			offerResponseParam := bitmark.NewTransferResponseParams(b, bitmark.Reject)
			offerResponseParam.Sign(p.Account)
			if _, err := bitmark.Respond(offerResponseParam); err != nil {
				return nil, err
			}

			switch fromcase {
			case ProcessReceivingTrialBitmarkFromMatchingService:
				fmt.Printf("%s rejected consent bitmark for %s from %s.\n", p.Name, referencedAssets[b.AssetID].Name, p.Identities[b.Offer.From])
			case ProcessReceivingTrialBitmarkFromSponsor:
				fmt.Printf("%s has opted to reject acceptance of consent bitmark from %s and refused the invitation to participate in %s.\n", p.Name, p.Identities[b.Offer.From], referencedAssets[b.AssetID].Name)
			}
		}
	}

	p.HoldingConsentBitmarkIDs = bitmarkIDs

	return bitmarkIDs, nil
}

func (p *Participant) SendBackTrialBitmark() error {
	for consentBitmarkID, medicalBitmarkID := range p.IssuedMedicalData {
		consentBitmarkInfo, err := bitmark.Get(consentBitmarkID)
		if err != nil {
			return err
		}

		matchingServiceAccountNumber := consentBitmarkInfo.Issuer

		// Transfer medical bitmark
		medicalOfferParam, _ := bitmark.NewOfferParams(matchingServiceAccountNumber, nil)
		medicalOfferParam.FromBitmark(medicalBitmarkID)
		medicalOfferParam.Sign(p.Account)
		if err := bitmark.Offer(medicalOfferParam); err != nil {
			return err
		}

		// Also transfer the consent bitmark
		consentOfferParam, _ := bitmark.NewOfferParams(matchingServiceAccountNumber, nil)
		consentOfferParam.FromBitmark(consentBitmarkID)
		consentOfferParam.Sign(p.Account)
		if err := bitmark.Offer(consentOfferParam); err != nil {
			return err
		}

		// Get bitmark info of trial
		medicalBitmarkInfo, err := bitmark.Get(medicalBitmarkID)
		if err != nil {
			return err
		}

		medicalAsset, err := asset.Get(medicalBitmarkInfo.AssetID)
		if err != nil {
			return err
		}

		identityForReceiver := p.Identities[matchingServiceAccountNumber]

		fmt.Printf("%s issued health data bitmark for %s and sent it to %s for evaluation along with consent bitmark.\n", p.Name, medicalAsset.Name, identityForReceiver)
	}
	return nil
}

func (p *Participant) IssueMedicalDataBitmark() ([]string, error) {
	medicalBitmarkIDs := make([]string, 0)
	for _, consentBitmarkID := range p.HoldingConsentBitmarkIDs {
		if !util.RandWithProb(p.conf.SubmitDataProb) {
			continue
		}

		consentBitmarkInfo, err := bitmark.Get(consentBitmarkID)
		if err != nil {
			return nil, err
		}
		consentAsset, err := asset.Get(consentBitmarkInfo.AssetID)
		if err != nil {
			return nil, err
		}

		medicalContent := "MEDICAL DATA\n" + util.RandStringBytesMaskImprSrc(1000)

		assetParam, err := asset.NewRegistrationParams(
			"health_data_"+p.Name+"_"+p.Identities[consentAsset.Registrant],
			map[string]string{
				"Type":          "Health Data",
				"Trial Bitmark": consentBitmarkID,
			},
		)

		assetParam.SetFingerprintFromData([]byte(medicalContent))
		assetParam.Sign(p.Account)
		assetID, err := asset.Register(assetParam)
		if err != nil {
			return nil, err
		}

		issueParam, _ := bitmark.NewIssuanceParams(
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
