package matchingservice

import (
	"fmt"

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

func (m *MatchingService) print(a ...interface{}) {
	c.Println("["+m.Name+"] ", a)
}
