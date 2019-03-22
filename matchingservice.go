package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmark-sdk-go/account"
	"github.com/bitmark-inc/bitmark-sdk-go/asset"
	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"
	"github.com/bitmark-inc/pfizer/util"
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
					issueParam := bitmark.NewIssuanceParams(assetID, 1)
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
		offerParam := bitmark.NewOfferParams(pp.Account.AccountNumber(), nil)
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

	bitmarks, err := bitmark.List(builder)
	if err != nil {
		return nil, err
	}

	bitmarkIDs := make([]string, 0)

	for _, b := range bitmarks {
		params := bitmark.NewTransferResponseParams(b, bitmark.Accept)
		params.Sign(m.Account)
		if err := bitmark.Respond(params); err != nil {
			return nil, err
		}

		bitmarkIDs = append(bitmarkIDs, b.Id)

		assetType, ok := b.Asset.Metadata["Type"]
		if ok {
			switch assetType {
			case "Trial":
				fmt.Printf("%s signed for acceptance of consent data bitmark for %s from %s.\n", m.Name, b.Asset.Name, m.Identities[b.Offer.From])
			case "Health Data":
				fmt.Printf("%s signed for acceptance of health data bitmark for %s from %s and is evaluating it.\n", m.Name, b.Asset.Name, m.Identities[b.Offer.From])
			default:
				fmt.Println("Unknow bitmark")
			}
		}
	}

	// txs := make(map[string]string)
	// for trialOfferID, medicalOfferID := range offerIDs {
	// 	// Accept trial offer id
	// 	trialTransferOffer, err := m.apiClient.GetTransferOfferById(trialOfferID)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if trialTransferOffer.To == m.Account.AccountNumber() {
	// 		trialBitmarkInfo, err := bitmark.Get(trialTransferOffer.BitmarkId, true)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		trialTxID, err := util.TryToActionTransfer(trialTransferOffer, "accept", m.Account, m.apiClient)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		fmt.Printf("%s signed for acceptance of consent data bitmark for %s from %s.\n", m.Name, trialBitmarkInfo.Asset.Name, m.Identities[trialTransferOffer.From])

	// 		// Accept medical offer id
	// 		medicalTransferOffer, err := m.apiClient.GetTransferOfferById(medicalOfferID)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		medicalTxID, err := util.TryToActionTransfer(medicalTransferOffer, "accept", m.Account, m.apiClient)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		medicalBitmarkInfo, err := bitmark.Get(medicalTransferOffer.BitmarkId, true)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		txs[trialTxID] = medicalTxID

	// 		fmt.Printf("%s signed for acceptance of health data bitmark for %s from %s and is evaluating it.\n", m.Name, trialBitmarkInfo.Asset.Name, m.Identities[medicalBitmarkInfo.Asset.Registrant])

	// 	}

	// }

	return bitmarkIDs, nil
}

func (m *MatchingService) EvaluateTrialFromParticipant() error {
	// Query all owning bitmarks
	builder := bitmark.NewQueryParamsBuilder().
		OwnedBy(m.Account.AccountNumber(), false).
		LoadAsset(true)

	bitmarks, err := bitmark.List(builder)
	if err != nil {
		return err
	}

	for _, b := range bitmarks {
		assetType, ok := b.Asset.Metadata["Type"]
		if ok && assetType == "Health Data" {
			consentBitmarkID, ok := b.Asset.Metadata["Trial Bitmark"]
			if !ok {
				continue // Continue if cannot find consent bitmark
			}

			consentBitmark, err := bitmark.Get(consentBitmarkID, true)
			if err != nil {
				return err
			}

			if util.RandWithProb(m.conf.MatchDataApprovalProb) {
				// Send to sponsor with two signatures transfer
				sponsorAccountNumber := consentBitmark.Asset.Registrant

				// Transfer medical bitmark
				medicalOfferParam := bitmark.NewOfferParams(sponsorAccountNumber, nil)
				medicalOfferParam.FromBitmark(b.Id)
				medicalOfferParam.Sign(m.Account)
				if err := bitmark.Offer(medicalOfferParam); err != nil {
					return err
				}

				// Also transfer the consent bitmark
				consentOfferParam := bitmark.NewOfferParams(sponsorAccountNumber, nil)
				consentOfferParam.FromBitmark(consentBitmark.Id)
				consentOfferParam.Sign(m.Account)
				if err := bitmark.Offer(consentOfferParam); err != nil {
					return err
				}

				fmt.Printf("%s approved health data bitmark for %s and sent it to %s for evaluation.\n", m.Name, b.Asset.Name, m.Identities[sponsorAccountNumber])
				fmt.Printf("%s sent consent bitmark for %s to %s.\n", m.Name, consentBitmark.Asset.Name, m.Identities[sponsorAccountNumber])
			} else {
				// Send to health data bitmark to participant with one signature transfer
				participantAccountNumber := b.Asset.Registrant

				medicalTransferParam := bitmark.NewTransferParams(participantAccountNumber)
				medicalTransferParam.FromBitmark(b.Id)
				medicalTransferParam.Sign(m.Account)
				_, err := bitmark.Transfer(medicalTransferParam)
				if err != nil {
					return err
				}

				// Send consent into trash bin account (all-zero pubkey account)
				consentTransferParam := bitmark.NewTransferParams(m.conf.TrashBinAccount)
				consentTransferParam.FromBitmark(consentBitmarkID)
				consentTransferParam.Sign(m.Account)
				_, err = bitmark.Transfer(consentTransferParam)
				if err != nil {
					return err
				}

				fmt.Printf("%s rejected health data bitmark for %s from %s. %s has sent the rejected health data bitmark back to %s.\n",
					m.Name,
					b.Asset.Name,
					m.Identities[b.Issuer],
					m.Name,
					m.Identities[b.Issuer])
			}
		}
	}

	// offerIDs := make(map[string]string)
	// for trialTx, medicalTx := range txs {
	// 	txInfo, err := tx.Get(trialTx)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if txInfo.Owner != m.Account.AccountNumber() {
	// 		continue
	// 	}

	// 	bitmarkInfo, err := bitmark.Get(txInfo.BitmarkID, true)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if util.RandWithProb(m.conf.MatchDataApprovalProb) {
	// 		// m.print("Accept the matching for tx: " + trialTx)

	// 		// Send bitmark to its asset's registrant
	// 		trialOfferID, err := util.TryToSubmitTransfer(txInfo.BitmarkID, bitmarkInfo.Asset.Registrant, m.Account, m.apiClient)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		// Get bitmark information to print out
	// 		medicalTxInfo, err := tx.Get(medicalTx)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		// Transfer also the medical data
	// 		medicalOfferID, err := util.TryToSubmitTransfer(medicalTxInfo.BitmarkID, bitmarkInfo.Asset.Registrant, m.Account, m.apiClient)

	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		offerIDs[trialOfferID] = medicalOfferID

	// 		fmt.Printf("%s approved health data bitmark for %s and sent it to %s for evaluation.\n", m.Name, bitmarkInfo.Asset.Name, m.Identities[bitmarkInfo.Asset.Registrant])
	// 		fmt.Printf("%s sent consent bitmark for %s to %s.\n", m.Name, bitmarkInfo.Asset.Name, m.Identities[bitmarkInfo.Asset.Registrant])
	// 	} else {
	// 		// m.print("Reject the matching for tx: " + trialTx)
	// 		// Get previous owner
	// 		previousTxInfo, err := tx.Get(txInfo.PreviousID)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		previousOwner := previousTxInfo.Owner

	// 		// Get bitmark id of medical tx
	// 		medicalTXInfo, err := tx.Get(medicalTx)

	// 		// Transfer bitmarks back to previous owner by one signature
	// 		// _, err = util.TryToTransferOneSignature(m.Account, txInfo.BitmarkID, previousOwner, m.apiClient)
	// 		// if err != nil {
	// 		// 	return nil, err
	// 		// }

	// 		_, err = util.TryToTransferOneSignature(m.Account, medicalTXInfo.BitmarkID, previousOwner, m.apiClient)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		// Get bitmark information to print out
	// 		medicalTxInfo, err := tx.Get(medicalTx)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		medicalBitmarkInfo, err := bitmark.Get(medicalTxInfo.BitmarkID, true)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		fmt.Printf("%s rejected health data bitmark for %s from %s. %s has sent the rejected health data bitmark back to %s.\n", m.Name, bitmarkInfo.Asset.Name, m.Identities[medicalBitmarkInfo.Bitmark.Issuer], m.Name, m.Identities[medicalBitmarkInfo.Bitmark.Issuer])
	// 	}
	// }

	return nil
}

func (m *MatchingService) print(a ...interface{}) {
	fmt.Println("["+m.Name+"] ", a)
}
