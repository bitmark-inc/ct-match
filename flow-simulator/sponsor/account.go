package sponsor

import (
	"net/http"
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
	Account    *sdk.Account
	Name       string
	apiClient  *sdk.Client
	conf       config.SponsorsConf
	Identities map[string]string
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
		bitmarkName := "bitmark_" + s.Name + "_t" + util.StringFromNum(i)
		trialContent := "CONSENT #" + strconv.Itoa(i) + "\n" + util.RandStringBytesMaskImprSrc(2000)
		af := sdk.NewAssetFile("asset_"+s.Name+"_t"+util.StringFromNum(i), []byte(trialContent), sdk.Public)
		bitmarkIDs, err := s.apiClient.IssueByAssetFile(s.Account, af, 1, &sdk.AssetInfo{
			Name: bitmarkName,
			Metadata: map[string]string{
				"Sponsor": s.Name,
			},
		})

		if err != nil {
			return nil, nil, err
		}

		// bitmarkID := bitmarkIDs[0]
		trialBitmarkIds = append(trialBitmarkIds, bitmarkIDs...)
		trialAssetIds = append(trialAssetIds, af.Id())
		// s.print("Issued trial bitmark from Sponsor: ", bitmarkID)
		c.Printf("%s announced a new trial by adding trial %s to the blockchain.\n", s.Name, bitmarkName)
	}

	return trialBitmarkIds, trialAssetIds, nil
}

func (s *Sponsor) AcceptTrialBackAndMedicalData(offerIDs map[string]string, network string, httpClient *http.Client) (map[string]string, error) {
	txs := make(map[string]string)
	for trialOfferID, medicalOfferID := range offerIDs {
		// Accept trial offer id
		trialTransferOffer, err := s.apiClient.GetTransferOfferById(trialOfferID)
		if err != nil {
			return nil, err
		}

		if trialTransferOffer.To == s.Account.AccountNumber() {
			trialCounterSign, err := trialTransferOffer.Record.Countersign(s.Account)
			if err != nil {
				return nil, err
			}

			trialTxID, err := s.apiClient.CompleteTransferOffer(s.Account, trialOfferID, "accept", trialCounterSign.Countersignature)

			// Accept medical offer id
			medicalTransferOffer, err := s.apiClient.GetTransferOfferById(medicalOfferID)
			if err != nil {
				return nil, err
			}

			medicalCounterSign, err := medicalTransferOffer.Record.Countersign(s.Account)
			if err != nil {
				return nil, err
			}

			medicalTxID, err := s.apiClient.CompleteTransferOffer(s.Account, medicalOfferID, "accept", medicalCounterSign.Countersignature)
			if err != nil {
				return nil, err
			}

			txs[trialTxID] = medicalTxID

			// Get bitmark info to print out
			trialBitmarkInfo, err := util.GetBitmarkInfo(trialTransferOffer.BitmarkId, network, httpClient)
			if err != nil {
				return nil, err
			}
			c.Printf("%s signed for acceptance of health data bitmark %s for trial %s from %s for %s and is evaluating it.\n", s.Name, medicalTransferOffer.BitmarkId, trialBitmarkInfo.Asset.Name, s.Identities[trialBitmarkInfo.Asset.Registrant])
			c.Printf("%s signed for acceptance of consent bitmark %s for trial %s from %s.\n", s.Name, trialTransferOffer.BitmarkId, trialBitmarkInfo.Asset.Name, s.Identities[trialBitmarkInfo.Asset.Registrant])
		}

	}

	return txs, nil
}

func (s *Sponsor) EvaluateTrialFromSponsor(txs map[string]string, network string, httpClient *http.Client) (map[string]string, error) {
	offerIDs := make(map[string]string)
	for trialTx, medicalTx := range txs {
		txInfo, err := util.GetTXInfo(trialTx, network, httpClient)
		if err != nil {
			return nil, err
		}

		if txInfo.Owner != s.Account.AccountNumber() {
			continue
		}

		bitmarkInfo, err := util.GetBitmarkInfo(txInfo.BitmarkID, network, httpClient)
		if err != nil {
			return nil, err
		}

		if util.RandWithProb(s.conf.DataApprovalProb) {
			// s.print("Accept the data for tx: " + trialTx)

			participantAccount := bitmarkInfo.Bitmark.Issuer

			// Send bitmark to its participant
			trialTransferOffer, err := sdk.NewTransferOffer(nil, trialTx, participantAccount, s.Account)
			if err != nil {
				return nil, err
			}

			trialOfferID, err := s.apiClient.SubmitTransferOffer(s.Account, trialTransferOffer, nil)
			if err != nil {
				return nil, err
			}

			offerIDs[trialOfferID] = participantAccount

			// Get bitmark id of medical tx
			medicalTXInfo, err := util.GetTXInfo(medicalTx, network, httpClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTXInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			c.Printf("%s approved health data bitmark %s for trial %s from %s for acceptance into trial %s and sent consent bitmark %s to %s for acceptance into trial %s.\n", s.Name, medicalTXInfo.BitmarkID, medicalBitmarkInfo.Asset.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant], medicalBitmarkInfo.Asset.Name, bitmarkInfo.Bitmark.ID, s.Identities[participantAccount], medicalBitmarkInfo.Asset.Name)
		} else {
			// s.print("Reject the data for tx: " + trialTx)
			// Get previous owner
			previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
			if err != nil {
				return nil, err
			}
			previousOwner := previousTxInfo.Owner

			// Get bitmark id of medical tx
			medicalTXInfo, err := util.GetTXInfo(medicalTx, network, httpClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTXInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			// Transfer bitmarks back to previous owner by one signature
			_, err = s.apiClient.Transfer(s.Account, txInfo.BitmarkID, previousOwner)
			if err != nil {
				return nil, err
			}

			_, err = s.apiClient.Transfer(s.Account, medicalTXInfo.BitmarkID, medicalBitmarkInfo.Asset.Registrant)
			if err != nil {
				return nil, err
			}

			c.Printf("%s rejected health data bitmark %s for trial %s from %s. %s has sent the rejected health data bitmark back to %s.\n", s.Name, medicalTXInfo.BitmarkID, bitmarkInfo.Asset.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant], s.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant])
		}
	}

	return offerIDs, nil
}
