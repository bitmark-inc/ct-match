package sponsor

import (
	"fmt"
	"net/http"

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
	index      int
	Name       string
	apiClient  *sdk.Client
	conf       config.SponsorsConf
	Identities map[string]string
}

func (s *Sponsor) print(a ...interface{}) {
	c.Println("["+s.Name+"] ", a)
}

func New(index int, name, seed string, client *sdk.Client, conf config.SponsorsConf) (*Sponsor, error) {
	acc, err := sdk.AccountFromSeed(seed)
	if err != nil {
		return nil, err
	}

	return &Sponsor{
		Account:   acc,
		apiClient: client,
		Name:      name,
		conf:      conf,
		index:     index,
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
		assetName := util.RandInPool(s.conf.StudiesPool)
		trialContent := assetName + "\n\n" + util.RandStringBytesMaskImprSrc(2000)
		af := sdk.NewAssetFile("asset_"+s.Name+"_t"+util.StringFromNum(i), []byte(trialContent), sdk.Public)
		bitmarkIDs, err := s.apiClient.IssueByAssetFile(s.Account, af, 1, &sdk.AssetInfo{
			Name: assetName,
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
		fmt.Printf("%s announced %s by adding the trial asset and bitmark to the blockchain.\n", s.Name, assetName)
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
			trialTxID, err := util.TryToActionTransfer(trialTransferOffer, "accept", s.Account, s.apiClient)
			if err != nil {
				return nil, err
			}

			// Accept medical offer id
			medicalTransferOffer, err := s.apiClient.GetTransferOfferById(medicalOfferID)
			if err != nil {
				return nil, err
			}

			medicalTxID, err := util.TryToActionTransfer(medicalTransferOffer, "accept", s.Account, s.apiClient)
			if err != nil {
				return nil, err
			}

			txs[trialTxID] = medicalTxID

			// Get bitmark info to print out
			trialBitmarkInfo, err := util.GetBitmarkInfo(trialTransferOffer.BitmarkId, network, httpClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTransferOffer.BitmarkId, network, httpClient)
			if err != nil {
				return nil, err
			}

			fmt.Printf("%s signed for acceptance of health data bitmark for %s from %s for %s and is evaluating it.\n", s.Name, trialBitmarkInfo.Asset.Name, s.Identities[trialTransferOffer.From], s.Identities[medicalBitmarkInfo.Asset.Registrant])
			fmt.Printf("%s signed for acceptance of consent bitmark for %s from %s.\n", s.Name, trialBitmarkInfo.Asset.Name, s.Identities[trialBitmarkInfo.Bitmark.Issuer])
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
			// Get bitmark id of medical tx
			medicalTXInfo, err := util.GetTXInfo(medicalTx, network, httpClient)
			if err != nil {
				return nil, err
			}

			medicalBitmarkInfo, err := util.GetBitmarkInfo(medicalTXInfo.BitmarkID, network, httpClient)
			if err != nil {
				return nil, err
			}

			participantAccount := medicalBitmarkInfo.Bitmark.Issuer

			// Send bitmark to its participant
			trialOfferID, err := util.TryToSubmitTransfer(txInfo.BitmarkID, participantAccount, s.Account, s.apiClient)
			if err != nil {
				return nil, err
			}

			offerIDs[trialOfferID] = participantAccount

			fmt.Printf("%s approved health data bitmark for %s from %s for acceptance into %s and sent consent bitmark to %s for acceptance into %s.\n", s.Name, bitmarkInfo.Asset.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant], bitmarkInfo.Asset.Name, s.Identities[participantAccount], bitmarkInfo.Asset.Name)
		} else {
			// s.print("Reject the data for tx: " + trialTx)
			// Get previous owner
			// previousTxInfo, err := util.GetTXInfo(txInfo.PreviousID, network, httpClient)
			// if err != nil {
			// 	return nil, err
			// }
			// previousOwner := previousTxInfo.Owner

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
			// _, err = util.TryToTransferOneSignature(s.Account, txInfo.BitmarkID, previousOwner, s.apiClient)
			// if err != nil {
			// 	return nil, err
			// }

			_, err = util.TryToTransferOneSignature(s.Account, medicalTXInfo.BitmarkID, medicalBitmarkInfo.Asset.Registrant, s.apiClient)
			if err != nil {
				return nil, err
			}

			fmt.Printf("%s rejected health data bitmark for %s from %s. %s has sent the rejected health data bitmark back to %s.\n", s.Name, bitmarkInfo.Asset.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant], s.Name, s.Identities[medicalBitmarkInfo.Asset.Registrant])
		}
	}

	return offerIDs, nil
}
