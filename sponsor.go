package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"

	"github.com/bitmark-inc/bitmark-sdk-go/account"
	"github.com/bitmark-inc/bitmark-sdk-go/asset"
	"github.com/bitmark-inc/ct-match/util"
)

type Sponsor struct {
	Account                        account.Account
	index                          int
	Name                           string
	conf                           SponsorsConf
	receivedTrialAndHealthBitmarks []*bitmark.Bitmark
	Identities                     map[string]string
}

func (s *Sponsor) print(a ...interface{}) {
	fmt.Println("["+s.Name+"] ", a)
}

func newSponsor(index int, name, seed string, conf SponsorsConf) (*Sponsor, error) {
	acc, err := account.FromSeed(seed)
	if err != nil {
		return nil, err
	}

	return &Sponsor{
		Account: acc,
		Name:    name,
		conf:    conf,
		index:   index,
	}, nil
}

// type TrialBitmark struct {
// 	BitmarkID string
// 	AssetID   string
// }

func (s *Sponsor) RegisterNewTrial() ([]string, []string, error) {
	numberOfTrials := util.RandWithRange(s.conf.TrialPerSponsorMin, s.conf.TrialPerSponsorMax)
	trialBitmarkIds := make([]string, 0)
	trialAssetIds := make([]string, 0)

	for i := 0; i < numberOfTrials; i++ {
		assetName := util.RandInPool(s.conf.StudiesPool)
		trialContent := assetName + "\n\n" + util.RandStringBytesMaskImprSrc(2000)
		assetParam, err := asset.NewRegistrationParams(
			assetName,
			map[string]string{
				"Sponsor": s.Name,
				"Type":    "Trial",
			},
		)
		assetParam.SetFingerprint([]byte(trialContent))
		assetParam.Sign(s.Account)
		assetID, err := asset.Register(assetParam)
		if err != nil {
			return nil, nil, err
		}

		issueParam := bitmark.NewIssuanceParams(
			assetID,
			1,
		)
		issueParam.Sign(s.Account)
		bitmarkIDs, err := bitmark.Issue(issueParam)
		if err != nil {
			return nil, nil, err
		}

		trialBitmarkIds = append(trialBitmarkIds, bitmarkIDs...)
		trialAssetIds = append(trialAssetIds, assetID)

		fmt.Printf("%s announced %s by adding the trial asset and bitmark to the blockchain.\n", s.Name, assetName)
	}

	return trialBitmarkIds, trialAssetIds, nil
}

func (s *Sponsor) AcceptTrialBackAndMedicalData() ([]string, error) {
	builder := bitmark.NewQueryParamsBuilder().
		OfferTo(s.Account.AccountNumber()).
		LoadAsset(true)

	bitmarks, err := bitmark.List(builder)
	if err != nil {
		return nil, err
	}
	bitmarkIDs := make([]string, 0)
	filterredBitmarks := make([]*bitmark.Bitmark, 0)

	for _, b := range bitmarks {
		params := bitmark.NewTransferResponseParams(b, bitmark.Accept)
		params.Sign(s.Account)
		if err := bitmark.Respond(params); err != nil {
			return nil, err
		}

		assetType, ok := b.Asset.Metadata["Type"]
		if ok {
			switch assetType {
			case "Trial":
				fmt.Printf("%s signed for acceptance of consent bitmark for %s from %s.\n",
					s.Name,
					b.Asset.Name,
					s.Identities[b.Offer.From])
				bitmarkIDs = append(bitmarkIDs, b.Id)
				filterredBitmarks = append(filterredBitmarks, b)
			case "Health Data":
				fmt.Printf("%s signed for acceptance of health data bitmark for %s from %s for %s and is evaluating it.\n",
					s.Name,
					b.Asset.Name,
					s.Identities[b.Offer.From],
					s.Identities[b.Asset.Registrant])
				bitmarkIDs = append(bitmarkIDs, b.Id)
				filterredBitmarks = append(filterredBitmarks, b)
			default:
				fmt.Println("Unknow bitmark")
			}
		}

	}

	s.receivedTrialAndHealthBitmarks = filterredBitmarks
	return bitmarkIDs, nil
}

func (s *Sponsor) EvaluateTrialFromSponsor() error {
	for _, b := range s.receivedTrialAndHealthBitmarks {
		assetType, ok := b.Asset.Metadata["Type"]
		if ok && assetType == "Health Data" {
			consentBitmarkID, ok := b.Asset.Metadata["Trial Bitmark"]
			if !ok {
				continue
			}

			consentBitmark, err := bitmark.Get(consentBitmarkID, true)
			if err != nil {
				return err
			}

			participantAccountNumber := b.Asset.Registrant

			if util.RandWithProb(s.conf.DataApprovalProb) {
				consentOfferParam := bitmark.NewOfferParams(participantAccountNumber, nil)
				consentOfferParam.FromBitmark(consentBitmarkID)
				consentOfferParam.Sign(s.Account)
				if err := bitmark.Offer(consentOfferParam); err != nil {
					return err
				}
				fmt.Printf("%s approved health data bitmark for %s from %s for acceptance into %s and sent consent bitmark to %s for acceptance into %s.\n", s.Name, b.Asset.Name, s.Identities[participantAccountNumber], b.Asset.Name, s.Identities[participantAccountNumber], consentBitmark.Asset.Name)
			} else {
				healthTransferParam := bitmark.NewTransferParams(participantAccountNumber)
				healthTransferParam.FromBitmark(b.Id)
				healthTransferParam.Sign(s.Account)
				_, err := bitmark.Transfer(healthTransferParam)
				if err != nil {
					return err
				}

				fmt.Printf("%s rejected health data bitmark for %s from %s. %s has sent the rejected health data bitmark back to %s.\n", s.Name, b.Asset.Name, s.Identities[participantAccountNumber], s.Name, s.Identities[participantAccountNumber])
			}
		}
	}

	return nil
}
