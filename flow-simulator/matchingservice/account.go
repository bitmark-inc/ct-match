package matchingservice

import (
	"fmt"
	"net/http"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/participant"
	"github.com/bitmark-inc/pfizer/flow-simulator/util"
	"github.com/fatih/color"
)

var (
	c   = color.New(color.FgBlue)
	tag = "[MATCHING_SERVICE] "
)

type MatchingService struct {
	Account             *sdk.Account
	apiClient           *sdk.Client
	Name                string
	conf                config.MatchingServiceConf
	issueMoreBitmarkIDs []string
}

func New(name, seed string, client *sdk.Client, conf config.MatchingServiceConf) (*MatchingService, error) {
	acc, err := sdk.AccountFromSeed(seed)
	if err != nil {
		return nil, err
	}

	c.Println(tag + "Initialize matching service with bitmark account: " + acc.AccountNumber())

	return &MatchingService{
		Account:   acc,
		apiClient: client,
		conf:      conf,
	}, nil
}

func (m *MatchingService) IssueMoreTrial(assetIDs []string) ([]string, error) {
	issueMoreBitmarkIDs := make([]string, 0)
	for _, assetID := range assetIDs {
		if util.RandWithProb(m.conf.SelectAssetProb) {
			fmt.Println("Issue more with assetID = ", assetID)
			bitmarkIDs, err := m.apiClient.IssueByAssetId(m.Account, assetID, 1)
			if err != nil {
				return nil, err
			}

			issueMoreBitmarkIDs = append(issueMoreBitmarkIDs, bitmarkIDs...)
			m.print("Issued more trial bitmark: ", bitmarkIDs[0])
		}
	}
	m.issueMoreBitmarkIDs = issueMoreBitmarkIDs

	return issueMoreBitmarkIDs, nil
}

func (m *MatchingService) SendTrialToParticipant(participantsList []*participant.Participant) ([]string, error) {
	transferOfferIDs := make([]string, 0)
	for _, issueMoreBitmarkID := range m.issueMoreBitmarkIDs {
		n := util.RandWithRange(0, len(participantsList)-1)
		pp := participantsList[n]

		transferOffer, err := sdk.NewTransferOffer(nil, issueMoreBitmarkID, pp.Account.AccountNumber(), m.Account)
		if err != nil {
			return nil, err
		}

		offerID, err := m.apiClient.SubmitTransferOffer(m.Account, transferOffer, nil)
		if err != nil {
			return nil, err
		}

		pp.AddTransferOffer(offerID)

		transferOfferIDs = append(transferOfferIDs, offerID)
	}
	return transferOfferIDs, nil
}

func (m *MatchingService) AcceptTrialBackAndMedicalData(offerIDs map[string]string) (map[string]string, error) {
	txs := make(map[string]string)
	for trialOfferID, medicalOfferID := range offerIDs {
		// Accept trial offer id
		trialTransferOffer, err := m.apiClient.GetTransferOfferById(trialOfferID)
		if err != nil {
			return nil, err
		}

		if trialTransferOffer.To == m.Account.AccountNumber() {
			trialCounterSign, err := trialTransferOffer.Record.Countersign(m.Account)
			if err != nil {
				return nil, err
			}

			trialTxID, err := m.apiClient.CompleteTransferOffer(m.Account, trialOfferID, "accept", trialCounterSign.Countersignature)

			// Accept medical offer id
			medicalTransferOffer, err := m.apiClient.GetTransferOfferById(medicalOfferID)
			if err != nil {
				return nil, err
			}

			medicalCounterSign, err := medicalTransferOffer.Record.Countersign(m.Account)
			if err != nil {
				return nil, err
			}

			medicalTxID, err := m.apiClient.CompleteTransferOffer(m.Account, medicalOfferID, "accept", medicalCounterSign.Countersignature)

			txs[trialTxID] = medicalTxID
		}

	}

	return txs, nil
}

func (m *MatchingService) EvaluateTrialFromParticipant(txs map[string]string, network string, httpClient *http.Client) (map[string]string, error) {
	offerIDs := make(map[string]string)
	for trialTx, medicalTx := range txs {
		txInfo, err := util.GetTXInfo(trialTx, network, httpClient)
		if err != nil {
			return nil, err
		}

		if txInfo.Owner != m.Account.AccountNumber() {
			continue
		}

		if util.RandWithProb(m.conf.MatchDataApprovalProb) {
			m.print("Accept the matching for tx: " + trialTx)

			bitmarkInfo, err := util.GetBitmarkInfo(txInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			// Send bitmark to its asset's registrant
			trialTransferOffer, err := sdk.NewTransferOffer(nil, trialTx, bitmarkInfo.Asset.Registrant, m.Account)
			if err != nil {
				return nil, err
			}

			trialOfferID, err := m.apiClient.SubmitTransferOffer(m.Account, trialTransferOffer, nil)
			if err != nil {
				return nil, err
			}

			// Transfer also the medical data
			medicalTransferOffer, err := sdk.NewTransferOffer(nil, medicalTx, bitmarkInfo.Asset.Registrant, m.Account)
			if err != nil {
				return nil, err
			}

			medicalOfferID, err := m.apiClient.SubmitTransferOffer(m.Account, medicalTransferOffer, nil)
			if err != nil {
				return nil, err
			}

			offerIDs[trialOfferID] = medicalOfferID
		} else {
			m.print("Reject the matching for tx: " + trialTx)
			// Get previous owner
			previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
			if err != nil {
				return nil, err
			}
			previousOwner := previousTxInfo.Owner

			// Get bitmark id of medical tx
			medicalTXInfo, err := util.GetTXInfo(medicalTx, network, httpClient)

			// Transfer bitmarks back to previous owner by one signature
			_, err = m.apiClient.Transfer(m.Account, txInfo.BitmarkID, previousOwner)
			if err != nil {
				return nil, err
			}

			_, err = m.apiClient.Transfer(m.Account, medicalTXInfo.BitmarkID, previousOwner)
			if err != nil {
				return nil, err
			}
		}
	}

	return offerIDs, nil
}

func (m *MatchingService) print(a ...interface{}) {
	c.Println("["+m.Name+"] ", a)
}
