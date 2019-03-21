package main

import (
	"net/http"
	"time"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/config"
	"github.com/bitmark-inc/pfizer/util"
)

type Simulator struct {
	conf       *config.Configuration
	sdkClient  *sdk.Client
	httpClient *http.Client

	matchingServices []*MatchingService
	participants     []*Participant
	sponsors         []*Sponsor
}

func newSimulator(conf *config.Configuration) *Simulator {
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
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
	identities := make(map[string]string)

	sponsors := make([]*Sponsor, 0)
	for i, account := range s.conf.Sponsors.Accounts {
		s, err := newSponsor(i, account.Identity, account.Seed, s.sdkClient, s.conf.Sponsors)
		if err != nil {
			return err
		}
		identities[s.Account.AccountNumber()] = s.Name
		sponsors = append(sponsors, s)
	}

	participants := make([]*Participant, 0)
	for i := 0; i < s.conf.Participants.ParticipantNum; i++ {
		pp, err := newParticipant(s.sdkClient, s.conf.Participants)
		if err != nil {
			return err
		}
		identities[pp.Account.AccountNumber()] = pp.Name
		participants = append(participants, pp)
	}

	matchingServices := make([]*MatchingService, 0)
	for _, account := range s.conf.MatchingService.Accounts {
		m, err := newMatchingService(account.Identity, account.Seed, s.sdkClient, s.conf.MatchingService)
		if err != nil {
			return err
		}

		m.Participants = participants

		identities[m.Account.AccountNumber()] = m.Name
		matchingServices = append(matchingServices, m)
	}

	// Add identities
	for _, ss := range sponsors {
		ss.Identities = identities
	}
	for _, ms := range matchingServices {
		ms.Identities = identities
	}
	for _, pp := range participants {
		pp.Identities = identities
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

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for bitmark to be confirmed
	util.WaitForBitmarkConfirmations(trialBitmarkIds, s.conf.Network, s.httpClient)

	// Issue more from matching service
	moreTrialBitmarkIDs := make([]string, 0)
	for _, ms := range matchingServices {
		bitmarkIDs, err := ms.IssueMoreTrial(trialAssetIds, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		moreTrialBitmarkIDs = append(moreTrialBitmarkIDs, bitmarkIDs...)
	}

	// Wait for bitmark to be confirmed
	util.WaitForBitmarkConfirmations(moreTrialBitmarkIDs, s.conf.Network, s.httpClient)

	// Send to participant
	for _, ms := range matchingServices {
		_, err := ms.SendTrialToParticipant(s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	//Ask for acceptance from participants
	sendToParticipantTxs := make([]string, 0)
	for _, pp := range participants {
		trialTXs, err := pp.ProcessRecevingTrialBitmark(ProcessReceivingTrialBitmarkFromMatchingService, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		sendToParticipantTxs = append(sendToParticipantTxs, trialTXs...)
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(sendToParticipantTxs, s.conf.Network, s.httpClient)

	// Issue medical data from participants that received the trial
	medicalBitmarkIDs := make([]string, 0)
	holdingConsentTxs := make([]string, 0)
	for _, pp := range participants {
		bitmarkIDs, err := pp.IssueMedicalDataBitmark(s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		medicalBitmarkIDs = append(medicalBitmarkIDs, bitmarkIDs...)
		holdingConsentTxs = append(holdingConsentTxs, pp.HoldingConsentTxs...)
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for bitmarks to be confirmed
	util.WaitForBitmarkConfirmations(medicalBitmarkIDs, s.conf.Network, s.httpClient)
	util.WaitForConfirmations(holdingConsentTxs, s.conf.Network, s.httpClient)

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

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept the medical data and trial from participants
	trialAndMedicalTxs := make(map[string]string)
	for _, ms := range matchingServices {
		txs, err := ms.AcceptTrialBackAndMedicalData(trialAndMedicalOfferIDs, s.conf.Network, s.httpClient)
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

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

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

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept receiving from sponsors
	acceptTrialAndMedicalFromSponsorTxs := make(map[string]string)
	for _, ss := range sponsors {
		txs, err := ss.AcceptTrialBackAndMedicalData(evaluationMatchingServiceOfferIDs, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		for k, v := range txs {
			acceptTrialAndMedicalFromSponsorTxs[k] = v
		}
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for transactions to be confirmed
	acceptTrialAndMedicalFromSponsorTxsInArray := make([]string, 0)
	for k, v := range acceptTrialAndMedicalFromSponsorTxs {
		acceptTrialAndMedicalFromSponsorTxsInArray = append(acceptTrialAndMedicalFromSponsorTxsInArray, k)
		acceptTrialAndMedicalFromSponsorTxsInArray = append(acceptTrialAndMedicalFromSponsorTxsInArray, v)
	}
	util.WaitForConfirmations(acceptTrialAndMedicalFromSponsorTxsInArray, s.conf.Network, s.httpClient)

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

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

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept transfer from participants
	sendFromSponsorToParticipantTxs := make([]string, 0)
	for _, pp := range participants {
		trialTXs, err := pp.ProcessRecevingTrialBitmark(ProcessReceivingTrialBitmarkFromSponsor, s.conf.Network, s.httpClient)
		if err != nil {
			return err
		}

		sendFromSponsorToParticipantTxs = append(sendFromSponsorToParticipantTxs, trialTXs...)
	}

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(sendFromSponsorToParticipantTxs, s.conf.Network, s.httpClient)

	return nil
}
