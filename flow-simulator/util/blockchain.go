package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func apiEndpoint(network string) string {
	if network == "livenet" {
		return "https://api.bitmark.com"
	} else {
		return "https://api.test.bitmark.com"
	}
}

func isTXConfirmed(tx string, network string, httpClient *http.Client) (bool, error) {
	url := apiEndpoint(network) + "/v1/txs/" + tx + "?pending=true"
	resp, err := httpClient.Get(url)
	if err != nil {
		return false, err
	}

	var data struct {
		TX struct {
			Status string `json:"status"`
		} `json:"tx"`
	}

	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	if err := decoder.Decode(&data); err != nil {
		return false, err
	}

	return data.TX.Status == "confirmed", nil
}

func isTXsConfirmed(txs []string, network string, httpClient *http.Client) bool {
	var wg sync.WaitGroup
	isConfirmedChan := make(chan bool, len(txs))

	wg.Add(len(txs))
	for _, tx := range txs {
		go func(tx string) {
			defer wg.Done()
			isConfirmed, _ := isTXConfirmed(tx, network, httpClient)
			isConfirmedChan <- isConfirmed
		}(tx)
	}

	wg.Wait()

	isConfirmed := true
	for {
		if len(isConfirmedChan) == 0 {
			return isConfirmed
		}

		isConfirmed, ok := <-isConfirmedChan
		if !ok {
			return isConfirmed
		}
		if isConfirmed == false {
			close(isConfirmedChan)
			return false
		}
	}
}

func WaitForConfirmation(tx string, network string, httpClient *http.Client) error {
	fmt.Println("Waiting for confirmations")
	for {
		confirmed, err := isTXConfirmed(tx, network, httpClient)
		if err != nil {
			return err
		}

		if confirmed {
			fmt.Println("Transaction is confirmed")
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func WaitForConfirmations(txs []string, network string, httpClient *http.Client) {
	fmt.Println("Waiting for confirmations")
	for {
		confirmed := isTXsConfirmed(txs, network, httpClient)

		if confirmed {
			fmt.Println("Transactions are confirmed")
			return
		}

		time.Sleep(1 * time.Second)
	}
}

type BitmarkTx struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	BlockNumber int    `json:"block_number"`
	Offset      int64  `json:"offset"`
	BitmarkID   string `json:"bitmark_id"`
	Status      string `json:"status"`
	PreviousID  string `json:"previous_id"`
}

func GetTXInfo(tx string, network string, httpClient *http.Client) (*BitmarkTx, error) {
	url := apiEndpoint(network) + "/v1/txs/" + tx + "?pending=true"
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	var data struct {
		Tx BitmarkTx `json:"tx"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data.Tx, nil
}

type Bitmark struct {
	ID          string    `json:"id"`
	HeadID      string    `json:"head_id"`
	Owner       string    `json:"owner"`
	Issuer      string    `json:"issuer"`
	IssuedAt    time.Time `json:"issued_at"`
	BlockNumber int       `json:"block_number"`
	Offset      int64     `json:"offset"`
	Status      string    `json:"status"`
}

type Asset struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Metadata   map[string]string `json:"metadata"`
	Registrant string            `json:"registrant"`
}

type BitmarkInfo struct {
	Bitmark Bitmark `json:"bitmark"`
	Asset   Asset   `json:"asset"`
}

func GetBitmarkInfo(bitmarkID string, network string, httpClient *http.Client) (*BitmarkInfo, error) {
	bitmarkInfoURL := apiEndpoint(network) + "/v1/bitmarks/" + bitmarkID + "?asset=true"
	resp, err := httpClient.Get(bitmarkInfoURL)
	if err != nil {
		return nil, err
	}

	var bitmarkInfo BitmarkInfo

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&bitmarkInfo); err != nil {
		return nil, err
	}

	return &bitmarkInfo, nil
}
