package simulator

import (
	"net/http"
	"strconv"
	"time"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/util"

	"github.com/bitmark-inc/pfizer/flow-simulator/matchingservice"
	"github.com/bitmark-inc/pfizer/flow-simulator/participant"
	"github.com/bitmark-inc/pfizer/flow-simulator/sponsor"
)

type Simulator struct {
	conf       *config.Configuration
	sdkClient  *sdk.Client
	httpClient *http.Client

	matchingServices []*matchingservice.MatchingService
	participants     []*participant.Participant
	sponsors         []*sponsor.Sponsor
}

func New(conf *config.Configuration) *Simulator {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	c := sdk.NewClient(&sdk.Config{
		HTTPClient: httpClient,
		Network:    conf.Network,
	})

	return &Simulator{
		conf:       conf,
		sdkClient:  c,
		httpClient: httpClient,
	}
}

func (s *Simulator) Simulate() error {
	matchingServices := make([]*matchingservice.MatchingService, 0)
	for i, seed := range s.conf.MatchingService.Accounts {
		m, err := matchingservice.New("MATCHING SERVICE "+strconv.Itoa(i), seed, s.sdkClient, s.conf.MatchingService)
		if err != nil {
			return err
		}
		matchingServices = append(matchingServices, m)
	}

	sponsors := make([]*sponsor.Sponsor, 0)
	for i, seed := range s.conf.Sponsors.Accounts {
		s, err := sponsor.New("SPONSOR "+strconv.Itoa(i), seed, s.sdkClient, s.conf.Sponsors)
		if err != nil {
			return err
		}
		sponsors = append(sponsors, s)
	}

	participants := make([]*participant.Participant, 0)
	for i := 1; i < s.conf.Participants.ParticipantNum; i++ {
		pp, err := participant.New("PARTICIPANT "+strconv.Itoa(i), s.sdkClient, s.conf.Participants)
		if err != nil {
			return err
		}
		participants = append(participants, pp)
	}

	// Register trial bitmark from sponsor
	trialBitmarkIds := make([]string, 0)
	trialAssetIds := make([]string, 0)

	for _, ss := range sponsors {
		bitmarkIds, assetIds, err := ss.RegisterNewTrial()
		if err != nil {
			return err
		}

		trialBitmarkIds = append(trialBitmarkIds, bitmarkIds...)
		trialAssetIds = append(trialAssetIds, assetIds...)
	}

	// Wait for bitmark to be confirmed
	util.WaitForConfirmations(trialBitmarkIds, s.conf.Network, s.httpClient)

	// Sleep for 2 seconds (workaround)
	time.Sleep(5 * time.Second)

	// Issue more from matching service
	moreTrialBitmarkIDs := make([]string, 0)
	for _, ms := range matchingServices {
		bitmarkIDs, err := ms.IssueMoreTrial(trialAssetIds)
		if err != nil {
			return err
		}

		moreTrialBitmarkIDs = append(moreTrialBitmarkIDs, bitmarkIDs...)
	}

	// Wait for bitmark to be confirmed
	util.WaitForConfirmations(moreTrialBitmarkIDs, s.conf.Network, s.httpClient)

	// Send to participant
	for _, ms := range matchingServices {
		_, err := ms.SendTrialToParticipant(participants)
		if err != nil {
			return err
		}
	}

	//Ask for acceptance from participants
	sendToParticipantTxs := make([]string, 0)
	for _, pp := range participants {
		trialTXs, err := pp.ProcessRecevingTrialBitmark(participant.ProcessReceivingTrialBitmarkFromMatchingService)
		if err != nil {
			return err
		}

		sendToParticipantTxs = append(sendToParticipantTxs, trialTXs...)
	}

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(sendToParticipantTxs, s.conf.Network, s.httpClient)

	// Issue medical data from participants that received the trial
	medicalBitmarkIDs := make([]string, 0)
	for _, pp := range participants {
		bitmarkIDs, err := pp.IssueMedicalDataBitmark(s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		medicalBitmarkIDs = append(medicalBitmarkIDs, bitmarkIDs...)
	}

	// Wait for bitmarks to be confirmed
	util.WaitForConfirmations(medicalBitmarkIDs, s.conf.Network, s.httpClient)

	// Send back the trial bitmark and medical data to matching service
	trialAndMedicalOfferIDs := make(map[string]string)
	for _, pp := range participants {
		offerIDs, err := pp.SendBackTrialBitmark(s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		for k, v := range offerIDs {
			trialAndMedicalOfferIDs[k] = v
		}
	}

	// Accept the medical data and trial from participants
	trialAndMedicalTxs := make(map[string]string)
	for _, ms := range matchingServices {
		txs, err := ms.AcceptTrialBackAndMedicalData(trialAndMedicalOfferIDs)
		if err != nil {
			return err
		}

		for k, v := range txs {
			trialAndMedicalTxs[k] = v
		}
	}

	// Wait for bitmarks to be confirmed
	trialAndMedicalTxsInArray := make([]string, 0)
	for k, v := range trialAndMedicalTxs {
		trialAndMedicalTxsInArray = append(trialAndMedicalTxsInArray, k)
		trialAndMedicalTxsInArray = append(trialAndMedicalTxsInArray, v)
	}
	util.WaitForConfirmations(trialAndMedicalTxsInArray, s.conf.Network, s.httpClient)

	// Evaluate the trial from participants
	evaluationMatchingServiceOfferIDs := make(map[string]string)
	for _, ms := range matchingServices {
		txs, err := ms.EvaluateTrialFromParticipant(trialAndMedicalTxs, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		for k, v := range txs {
			evaluationMatchingServiceOfferIDs[k] = v
		}
	}

	// Accept receiving from sponsors
	acceptTrialAndMedicalFromSponsorTxs := make(map[string]string)
	acceptTrialAndMedicalFromSponsorTxsInArray := make([]string, 0)
	for _, ss := range sponsors {
		txs, err := ss.AcceptTrialBackAndMedicalData(evaluationMatchingServiceOfferIDs)
		if err != nil {
			return err
		}

		for k, v := range txs {
			acceptTrialAndMedicalFromSponsorTxs[k] = v
			acceptTrialAndMedicalFromSponsorTxsInArray = append(acceptTrialAndMedicalFromSponsorTxsInArray, k)
			acceptTrialAndMedicalFromSponsorTxsInArray = append(acceptTrialAndMedicalFromSponsorTxsInArray, v)
		}
	}

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(acceptTrialAndMedicalFromSponsorTxsInArray, s.conf.Network, s.httpClient)

	// Evaluate from sponsors
	for _, ss := range sponsors {
		offerIDs, err := ss.EvaluateTrialFromSponsor(acceptTrialAndMedicalFromSponsorTxs, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		for offerID, participantAccount := range offerIDs {
			for _, pp := range participants {
				if participantAccount == pp.Account.AccountNumber() {
					pp.AddTransferOffer(offerID)
					break
				}
			}
		}
	}

	// Accept transfer from participants
	sendFromSponsorToParticipantTxs := make([]string, 0)
	for _, pp := range participants {
		trialTXs, err := pp.ProcessRecevingTrialBitmark(participant.ProcessReceivingTrialBitmarkFromSponsor)
		if err != nil {
			return err
		}

		sendFromSponsorToParticipantTxs = append(sendFromSponsorToParticipantTxs, trialTXs...)
	}

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(sendFromSponsorToParticipantTxs, s.conf.Network, s.httpClient)

	return nil
}
