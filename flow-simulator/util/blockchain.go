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
	if err := decoder.Decode(&data); err != nil {
		return false, err
	}

	return data.TX.Status == "confirmed", nil
}

func isTXsConfirmed(txs []string, network string, httpClient *http.Client) bool {
	var wg sync.WaitGroup
	isConfirmedChan := make(chan bool)

	wg.Add(len(txs))
	for _, tx := range txs {
		go func(tx string) {
			defer wg.Done()

			isConfirmed, _ := isTXConfirmed(tx, network, httpClient)
			isConfirmedChan <- isConfirmed
		}(tx)
	}

	isConfirmed := true
	go func() {
		for i := range isConfirmedChan {
			if i == false {
				isConfirmed = false
			}
		}
	}()

	wg.Wait()

	return isConfirmed
}

func WaitForConfirmation(tx string, network string, httpClient *http.Client) error {
	fmt.Println("Waiting for comfirmation for tx: ", tx)
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
	fmt.Printf("Waiting for comfirmation for tx: %+v\n", txs)
	for {
		confirmed := isTXsConfirmed(txs, network, httpClient)

		if confirmed {
			fmt.Println("Transaction is confirmed")
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
