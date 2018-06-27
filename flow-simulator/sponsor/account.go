package sponsor

import (
	"strconv"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/util"
	"github.com/fatih/color"
)

var (
	c = color.New(color.FgYellow)
)

type Sponsor struct {
	Account   *sdk.Account
	Name      string
	apiClient *sdk.Client
	conf      config.SponsorsConf
}

func (s *Sponsor) print(a ...interface{}) {
	c.Println("["+s.Name+"] ", a)
}

func New(name, seed string, client *sdk.Client, conf config.SponsorsConf) (*Sponsor, error) {
	acc, err := sdk.AccountFromSeed(seed)
	if err != nil {
		return nil, err
	}

	c.Println("Initialize sponsor with bitmark account: " + acc.AccountNumber())

	return &Sponsor{
		Account:   acc,
		apiClient: client,
		Name:      name,
		conf:      conf,
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
		trialContent := "CONSENT #" + strconv.Itoa(i) + "\n" + util.RandStringBytesMaskImprSrc(2000)
		af := sdk.NewAssetFile("trial.txt", []byte(trialContent), sdk.Public)
		bitmarkIDs, err := s.apiClient.IssueByAssetFile(s.Account, af, 1, &sdk.AssetInfo{
			Name: "Trial from Sponsor #" + strconv.Itoa(i),
			Metadata: map[string]string{
				"Sponsor": s.Name,
			},
		})

		if err != nil {
			return nil, nil, err
		}

		bitmarkID := bitmarkIDs[0]
		trialBitmarkIds = append(trialBitmarkIds, bitmarkIDs...)
		trialAssetIds = append(trialAssetIds, af.Id())
		s.print("Issued trial bitmark from Sponsor: ", bitmarkID)
	}

	return trialBitmarkIds, trialAssetIds, nil
}
