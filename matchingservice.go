package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmark-sdk-go/account"
	"github.com/bitmark-inc/bitmark-sdk-go/asset"
	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"
	"github.com/bitmark-inc/ct-match/util"
)

type MatchingService struct {
	Account             account.Account
	Name                string
	conf                MatchingServiceConf
	Participants        []*Participant
	issueMoreBitmarkIDs map[string]*Participant
	Identities          map[string]string
}

func newMatchingService(name, seed string, conf MatchingServiceConf) (*MatchingService, error) {
	acc, err := account.FromSeed(seed)
	if err != nil {
		return nil, err
	}

	// fmt.Println(tag + "Initialize matching service with bitmark account: " + acc.AccountNumber())

	return &MatchingService{
		Account:             acc,
		conf:                conf,
		Name:                name,
		issueMoreBitmarkIDs: make(map[string]*Participant),
	}, nil
}

func (m *MatchingService) IssueMoreTrial(assetIDs []string) ([]string, error) {
	totalBitmarkIDs := make([]string, 0)
	for _, assetID := range assetIDs {
		assetInfo, err := asset.Get(assetID)
		if err != nil {
			return nil, err
		}

		if util.RandWithProb(m.conf.SelectAssetProb) {
			for _, p := range m.Participants {
				if util.RandWithProb(m.conf.MatchProb) {
					issueParam, _ := bitmark.NewIssuanceParams(assetID, 1)
					issueParam.Sign(m.Account)

					bitmarkIDs, err := bitmark.Issue(issueParam)
					if err != nil {
						return nil, err
					}

					bitmarkID := bitmarkIDs[0]
					totalBitmarkIDs = append(totalBitmarkIDs, bitmarkID)
					m.issueMoreBitmarkIDs[bitmarkID] = p
					fmt.Printf("%s considered %s for %s and found a match. %s issued consent bitmark for %s and sent it to %s for acceptance.\n", m.Name, p.Name, assetInfo.Name, m.Name, assetInfo.Name, p.Name)
				} else {
					fmt.Printf("%s considered %s for %s and found no match.\n", m.Name, p.Name, assetInfo.Name)
				}
			}
		}
	}

	return totalBitmarkIDs, nil
}

func (m *MatchingService) SendTrialToParticipant() error {
	for issueMoreBitmarkID, pp := range m.issueMoreBitmarkIDs {
		offerParam, _ := bitmark.NewOfferParams(pp.Account.AccountNumber(), nil)
		offerParam.FromBitmark(issueMoreBitmarkID)
		offerParam.Sign(m.Account)
		if err := bitmark.Offer(offerParam); err != nil {
			return err
		}
	}

	return nil
}

func (m *MatchingService) AcceptTrialBackAndMedicalData() ([]string, error) {
	builder := bitmark.NewQueryParamsBuilder().
		OfferTo(m.Account.AccountNumber()).
		LoadAsset(true)

	bitmarks, assets, err := bitmark.List(builder)
	if err != nil {
		return nil, err
	}
	referencedAssets := make(map[string]*asset.Asset)
	for _, asset := range assets {
		referencedAssets[asset.ID] = asset
	}

	bitmarkIDs := make([]string, 0)

	for _, b := range bitmarks {
		params := bitmark.NewTransferResponseParams(b, bitmark.Accept)
		params.Sign(m.Account)
		if _, err := bitmark.Respond(params); err != nil {
			return nil, err
		}

		bitmarkIDs = append(bitmarkIDs, b.ID)

		assetType, ok := referencedAssets[b.AssetID].Metadata["Type"]
		if ok {
			switch assetType {
			case "Trial":
				fmt.Printf("%s signed for acceptance of consent data bitmark for %s from %s.\n", m.Name, referencedAssets[b.AssetID].Name, m.Identities[b.Offer.From])
			case "Health Data":
				fmt.Printf("%s signed for acceptance of health data bitmark for %s from %s and is evaluating it.\n", m.Name, referencedAssets[b.AssetID].Name, m.Identities[b.Offer.From])
			default:
				fmt.Println("Unknow bitmark")
			}
		}
	}

	return bitmarkIDs, nil
}

func (m *MatchingService) EvaluateTrialFromParticipant() error {
	// Query all owning bitmarks
	builder := bitmark.NewQueryParamsBuilder().
		OwnedBy(m.Account.AccountNumber()).
		LoadAsset(true)

	bitmarks, assets, err := bitmark.List(builder)
	if err != nil {
		return err
	}
	referencedAssets := make(map[string]*asset.Asset)
	for _, asset := range assets {
		referencedAssets[asset.ID] = asset
	}

	for _, b := range bitmarks {
		assetType, ok := referencedAssets[b.AssetID].Metadata["Type"]
		if ok && assetType == "Health Data" {
			consentBitmarkID, ok := referencedAssets[b.AssetID].Metadata["Trial Bitmark"]
			if !ok {
				continue // Continue if cannot find consent bitmark
			}

			consentBitmark, err := bitmark.Get(consentBitmarkID)
			if err != nil {
				return err
			}
			consentAsset, err := asset.Get(consentBitmark.AssetID)
			if err != nil {
				return err
			}

			if util.RandWithProb(m.conf.MatchDataApprovalProb) {
				// Send to sponsor with two signatures transfer
				sponsorAccountNumber := referencedAssets[b.AssetID].Registrant

				// Transfer medical bitmark
				medicalOfferParam, _ := bitmark.NewOfferParams(sponsorAccountNumber, nil)
				medicalOfferParam.FromBitmark(b.ID)
				medicalOfferParam.Sign(m.Account)
				if err := bitmark.Offer(medicalOfferParam); err != nil {
					return err
				}

				// Also transfer the consent bitmark
				consentOfferParam, _ := bitmark.NewOfferParams(sponsorAccountNumber, nil)
				consentOfferParam.FromBitmark(consentBitmark.ID)
				consentOfferParam.Sign(m.Account)
				if err := bitmark.Offer(consentOfferParam); err != nil {
					return err
				}

				fmt.Printf("%s approved health data bitmark for %s and sent it to %s for evaluation.\n", m.Name, referencedAssets[b.AssetID].Name, m.Identities[sponsorAccountNumber])
				fmt.Printf("%s sent consent bitmark for %s to %s.\n", m.Name, consentAsset.Name, m.Identities[sponsorAccountNumber])
			} else {
				// Send to health data bitmark to participant with one signature transfer
				participantAccountNumber := referencedAssets[b.AssetID].Registrant

				medicalTransferParam, _ := bitmark.NewTransferParams(participantAccountNumber)
				medicalTransferParam.FromBitmark(b.ID)
				medicalTransferParam.Sign(m.Account)
				_, err := bitmark.Transfer(medicalTransferParam)
				if err != nil {
					return err
				}

				// Send consent into trash bin account (all-zero pubkey account)
				consentTransferParam, _ := bitmark.NewTransferParams(m.conf.TrashBinAccount)
				consentTransferParam.FromBitmark(consentBitmarkID)
				consentTransferParam.Sign(m.Account)
				_, err = bitmark.Transfer(consentTransferParam)
				if err != nil {
					return err
				}

				fmt.Printf("%s rejected health data bitmark for %s from %s. %s has sent the rejected health data bitmark back to %s.\n",
					m.Name,
					referencedAssets[b.AssetID].Name,
					m.Identities[b.Issuer],
					m.Name,
					m.Identities[b.Issuer])
			}
		}
	}

	return nil
}

func (m *MatchingService) print(a ...interface{}) {
	fmt.Println("["+m.Name+"] ", a)
}
