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
	Identities          map[string]string
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
		Name:      name,
	}, nil
}

func (m *MatchingService) IssueMoreTrial(assetIDs []string) ([]string, error) {
	issueMoreBitmarkIDs := make([]string, 0)
	for _, assetID := range assetIDs {
		if util.RandWithProb(m.conf.SelectAssetProb) {
			bitmarkIDs, err := m.apiClient.IssueByAssetId(m.Account, assetID, 1)
			if err != nil {
				return nil, err
			}

			issueMoreBitmarkIDs = append(issueMoreBitmarkIDs, bitmarkIDs...)
			// m.print("Issued more trial bitmark: ", bitmarkIDs[0])
		}
	}
	m.issueMoreBitmarkIDs = issueMoreBitmarkIDs

	return issueMoreBitmarkIDs, nil
}

func (m *MatchingService) SendTrialToParticipant(participantsList []*participant.Participant, network string, httpClient *http.Client) ([]string, error) {
	transferOfferIDs := make([]string, 0)
	for _, issueMoreBitmarkID := range m.issueMoreBitmarkIDs {
		bitmarkInfo, err := util.GetBitmarkInfo(issueMoreBitmarkID, network, httpClient)
		if err != nil {
			return nil, err
		}

		if !util.RandWithProb(m.conf.MatchProb) {
			fmt.Printf("%s considered P01 for trial %s and found no match.\n", m.Name, bitmarkInfo.Asset.Name)
			continue
		}

		n := util.RandWithRange(0, len(participantsList)-1)
		pp := participantsList[n]

		fmt.Printf("%s considered %s for trial %s and found a match. %s issued consent bitmark %s for trial %s and sent it to %s for acceptance.\n", m.Name, pp.Name, bitmarkInfo.Asset.Name, m.Name, bitmarkInfo.Bitmark.ID, bitmarkInfo.Asset.Name, pp.Name)

		offerID, err := util.TryToSubmitTransfer(issueMoreBitmarkID, pp.Account.AccountNumber(), m.Account, m.apiClient)

		if err != nil {
			return nil, err
		}

		pp.AddTransferOffer(offerID)

		transferOfferIDs = append(transferOfferIDs, offerID)
	}
	return transferOfferIDs, nil
}

func (m *MatchingService) AcceptTrialBackAndMedicalData(offerIDs map[string]string, network string, httpClient *http.Client) (map[string]string, error) {
	txs := make(map[string]string)
	for trialOfferID, medicalOfferID := range offerIDs {
		// Accept trial offer id
		trialTransferOffer, err := m.apiClient.GetTransferOfferById(trialOfferID)
		if err != nil {
			return nil, err
		}

		if trialTransferOffer.To == m.Account.AccountNumber() {
			trialBitmarkInfo, err := util.GetBitmarkInfo(trialTransferOffer.BitmarkId, network, httpClient)
			if err != nil {
				return nil, err
			}

			trialTxID, err := util.TryToActionTransfer(trialTransferOffer, "accept", m.Account, m.apiClient)
			if err != nil {
				return nil, err
			}

			fmt.Printf("%s signed for acceptance of consent data bitmark %s for trial %s from %s.\n", m.Name, trialTransferOffer.BitmarkId, trialBitmarkInfo.Asset.Name, m.Identities[trialTransferOffer.From])

			// Accept medical offer id
			medicalTransferOffer, err := m.apiClient.GetTransferOfferById(medicalOfferID)
			if err != nil {
				return nil, err
			}

			medicalTxID, err := util.TryToActionTransfer(medicalTransferOffer, "accept", m.Account, m.apiClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTransferOffer.BitmarkId, network, httpClient)
			if err != nil {
				return nil, err
			}

			txs[trialTxID] = medicalTxID

			fmt.Printf("%s signed for acceptance of health data bitmark %s for trial %s from %s and is evaluating it.\n", m.Name, medicalTransferOffer.BitmarkId, trialBitmarkInfo.Asset.Name, m.Identities[medicalBitmarkInfo.Asset.Registrant])

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
			// m.print("Accept the matching for tx: " + trialTx)

			bitmarkInfo, err := util.GetBitmarkInfo(txInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			// Send bitmark to its asset's registrant
			trialOfferID, err := util.TryToSubmitTransfer(txInfo.BitmarkID, bitmarkInfo.Asset.Registrant, m.Account, m.apiClient)
			if err != nil {
				return nil, err
			}

			// Get bitmark information to print out
			medicalTxInfo, err := util.GetTXInfo(medicalTx, network, httpClient)
			if err != nil {
				return nil, err
			}

			// Transfer also the medical data
			medicalOfferID, err := util.TryToSubmitTransfer(medicalTxInfo.BitmarkID, bitmarkInfo.Asset.Registrant, m.Account, m.apiClient)

			if err != nil {
				return nil, err
			}

			offerIDs[trialOfferID] = medicalOfferID

			fmt.Printf("%s approved health data bitmark %s for trial %s and sent it to %s for evaluation.\n", m.Name, medicalTxInfo.BitmarkID, bitmarkInfo.Asset.Name, m.Identities[bitmarkInfo.Asset.Registrant])
			fmt.Printf("%s sent consent bitmark %s for trial %s to %s.\n", m.Name, txInfo.BitmarkID, bitmarkInfo.Asset.Name, m.Identities[bitmarkInfo.Asset.Registrant])
		} else {
			// m.print("Reject the matching for tx: " + trialTx)
			// Get previous owner
			previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
			if err != nil {
				return nil, err
			}
			previousOwner := previousTxInfo.Owner

			// Get bitmark id of medical tx
			medicalTXInfo, err := util.GetTXInfo(medicalTx, network, httpClient)

			// Transfer bitmarks back to previous owner by one signature
			_, err = util.TryToTransferOneSignature(m.Account, txInfo.BitmarkID, previousOwner, m.apiClient)
			if err != nil {
				return nil, err
			}

			_, err = util.TryToTransferOneSignature(m.Account, medicalTXInfo.BitmarkID, previousOwner, m.apiClient)
			if err != nil {
				return nil, err
			}

			// Get bitmark information to print out
			medicalTxInfo, err := util.GetTXInfo(medicalTx, network, httpClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTxInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			fmt.Printf("%s rejected health data bitmark %s for trial %s from %s. %s has sent the rejected health data bitmark back to %s.\n", m.Name, medicalTxInfo.BitmarkID, medicalBitmarkInfo.Asset.Name, m.Identities[medicalBitmarkInfo.Bitmark.Issuer], m.Name, m.Identities[medicalBitmarkInfo.Bitmark.Issuer])
		}
	}

	return offerIDs, nil
}

func (m *MatchingService) print(a ...interface{}) {
	c.Println("["+m.Name+"] ", a)
}
