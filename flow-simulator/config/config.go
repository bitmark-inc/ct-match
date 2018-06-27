package config

import (
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl"
)

// Configuration is the main configuration structure

type MatchingServiceConf struct {
	Accounts              []string `hcl:"accounts"`
	SelectAssetProb       float64  `hcl:"select_asset_prob"`
	MatchProb             float64  `hcl:"match_prob"`
	MatchDataApprovalProb float64  `hcl:"match_data_approval_prob"`
}

type SponsorsConf struct {
	Accounts           []string `hcl:"accounts"`
	DataApprovalProb   float64  `hcl:"sponsor_data_approval_prob"`
	TrialPerSponsorMin int      `hcl:"trials_per_sponsor_min"`
	TrialPerSponsorMax int      `hcl:"trials_per_sponsor_max"`
}

type ParticipantsConf struct {
	ParticipantNum        int     `hcl:"participant_num"`
	AcceptMatchProb       float64 `hcl:"participant_accept_match_prob"`
	SubmitDataProb        float64 `hcl:"participant_submit_data_prob"`
	AcceptTrialInviteProb float64 `hcl:"participant_accept_trial_invite_prob"`
}

type Configuration struct {
	Network         string              `hcl:"network"`
	WaitTime        int                 `hcl:"wait_time"`
	MatchingService MatchingServiceConf `hcl:"matchingService"`
	Sponsors        SponsorsConf        `hcl:"sponsors"`
	Participants    ParticipantsConf    `hcl:"participants"`
}

// Load will read configuration from file
func Load(fileName string) (*Configuration, error) {
	var m Configuration
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err = hcl.Unmarshal(b, &m); nil != err {
		return nil, err
	}

	return &m, nil
}
