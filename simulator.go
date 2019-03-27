package main

import (
	"net/http"
	"time"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/util"
)

type Simulator struct {
	conf *Configuration

	matchingServices []*MatchingService
	participants     []*Participant
	sponsors         []*Sponsor
}

func newSimulator(conf *Configuration) *Simulator {
	return &Simulator{
		conf: conf,
	}
}

func (s *Simulator) Simulate() error {
	// Inititate go sdk
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	config := &sdk.Config{
		APIToken:   s.conf.APIToken,
		Network:    sdk.Network(s.conf.Network),
		HTTPClient: httpClient,
	}
	sdk.Init(config)

	identities := make(map[string]string)

	sponsors := make([]*Sponsor, 0)
	for i, account := range s.conf.Sponsors.Accounts {
		s, err := newSponsor(i, account.Identity, account.Seed, s.conf.Sponsors)
		if err != nil {
			return err
		}
		identities[s.Account.AccountNumber()] = s.Name
		sponsors = append(sponsors, s)
	}

	participants := make([]*Participant, 0)
	for i := 0; i < s.conf.Participants.ParticipantNum; i++ {
		pp, err := newParticipant(s.conf.Participants)
		if err != nil {
			return err
		}
		identities[pp.Account.AccountNumber()] = pp.Name
		participants = append(participants, pp)
	}

	matchingServices := make([]*MatchingService, 0)
	for _, account := range s.conf.MatchingService.Accounts {
		m, err := newMatchingService(account.Identity, account.Seed, s.conf.MatchingService)
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
	util.WaitForBitmarkConfirmations(trialBitmarkIds)

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
	util.WaitForBitmarkConfirmations(moreTrialBitmarkIDs)

	// Send to participant
	for _, ms := range matchingServices {
		err := ms.SendTrialToParticipant()
		if err != nil {
			return err
		}
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Ask for acceptance from participants
	sendToParticipantBitmarkIDs := make([]string, 0)
	for _, pp := range participants {
		trialBitmarkIDs, err := pp.ProcessRecevingTrialBitmark(ProcessReceivingTrialBitmarkFromMatchingService)
		if err != nil {
			return err
		}

		sendToParticipantBitmarkIDs = append(sendToParticipantBitmarkIDs, trialBitmarkIDs...)
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for transactions to be confirmed
	util.WaitForConfirmations(sendToParticipantBitmarkIDs)

	// Issue medical data from participants that received the trial
	medicalBitmarkIDs := make([]string, 0)
	holdingConsentBitmarkIDs := make([]string, 0)
	for _, pp := range participants {
		bitmarkIDs, err := pp.IssueMedicalDataBitmark()
		if err != nil {
			return err
		}

		medicalBitmarkIDs = append(medicalBitmarkIDs, bitmarkIDs...)
		holdingConsentBitmarkIDs = append(holdingConsentBitmarkIDs, pp.HoldingConsentBitmarkIDs...)
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Wait for bitmarks to be confirmed
	util.WaitForBitmarkConfirmations(medicalBitmarkIDs)
	util.WaitForBitmarkConfirmations(holdingConsentBitmarkIDs)

	// Send back the trial bitmark and medical data to matching service
	for _, pp := range participants {
		err := pp.SendBackTrialBitmark()
		if err != nil {
			return err
		}
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept the medical data and trial from participants
	trialAndMedicalBitmarkIDs := make([]string, 0)
	for _, ms := range matchingServices {
		bitmarkIDs, err := ms.AcceptTrialBackAndMedicalData()
		if err != nil {
			return err
		}

		trialAndMedicalBitmarkIDs = append(trialAndMedicalBitmarkIDs, bitmarkIDs...)
	}

	// Wait for bitmarks to be confirmed
	util.WaitForBitmarkConfirmations(trialAndMedicalBitmarkIDs)

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Evaluate the trial from participants
	for _, ms := range matchingServices {
		err := ms.EvaluateTrialFromParticipant()
		if err != nil {
			return err
		}
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept receiving from sponsors
	acceptTrialAndMedicalFromSponsorBitmarkIDs := make([]string, 0)
	for _, ss := range sponsors {
		bitmarkIDs, err := ss.AcceptTrialBackAndMedicalData()
		if err != nil {
			return err
		}

		acceptTrialAndMedicalFromSponsorBitmarkIDs = append(acceptTrialAndMedicalFromSponsorBitmarkIDs, bitmarkIDs...)
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	util.WaitForBitmarkConfirmations(acceptTrialAndMedicalFromSponsorBitmarkIDs)

	// Evaluate from sponsors
	for _, ss := range sponsors {
		err := ss.EvaluateTrialFromSponsor()
		if err != nil {
			return err
		}

		// for offerID, participantAccount := range offerIDs {
		// 	for _, pp := range participants {
		// 		if participantAccount == pp.Account.AccountNumber() {
		// 			pp.AddTransferOffer(offerID)
		// 			break
		// 		}
		// 	}
		// }
	}

	time.Sleep(time.Duration(s.conf.WaitTime) * time.Second)

	// Accept transfer from participants
	sendFromSponsorToParticipantTxs := make([]string, 0)
	for _, pp := range participants {
		trialTXs, err := pp.ProcessRecevingTrialBitmark(ProcessReceivingTrialBitmarkFromSponsor)
		if err != nil {
			return err
		}

		sendFromSponsorToParticipantTxs = append(sendFromSponsorToParticipantTxs, trialTXs...)
	}

	// Wait for transactions to be confirmed
	util.WaitForBitmarkConfirmations(sendFromSponsorToParticipantTxs)

	return nil
}
