package util

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"
	"github.com/bitmark-inc/bitmark-sdk-go/tx"
)

func apiEndpoint(network string) string {
	if network == "livenet" {
		return "https://api.bitmark.com"
	} else {
		return "https://api.test.bitmark.com"
	}
}

func isTXConfirmed(txID string) (bool, error) {
	result, err := tx.Get(txID, false)
	return result.Status == "confirmed", err
}

func isTXsConfirmed(txs []string) bool {
	var wg sync.WaitGroup
	isConfirmedChan := make(chan bool, len(txs))

	wg.Add(len(txs))
	for _, tx := range txs {
		go func(tx string) {
			defer wg.Done()
			isConfirmed, _ := isTXConfirmed(tx)
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

func WaitForConfirmation(tx string) error {
	fmt.Println("Waiting for confirmations")
	for {
		confirmed, err := isTXConfirmed(tx)
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

func WaitForConfirmations(txs []string) {
	fmt.Println("Waiting for confirmations")
	for {
		confirmed := isTXsConfirmed(txs)

		if confirmed {
			fmt.Println("Transactions are confirmed")
			return
		}

		time.Sleep(1 * time.Second)
	}
}

func isBitmarkConfirmed(bitmarkID string) (bool, error) {
	result, err := bitmark.Get(bitmarkID, false)
	if err != nil {
		fmt.Printf("Get bitmark id: %s, error: %v", bitmarkID, err)
		return false, err
	}
	return result.Status == "settled", nil
}

func filterUnconfirmedBitmarks(bitmarks []string) []string {
	var wg sync.WaitGroup
	bitmarkChan := make(chan *bitmark.Bitmark, len(bitmarks))

	wg.Add(len(bitmarks))
	for _, bitmarkID := range bitmarks {
		go func(bitmarkID string) {
			defer wg.Done()
			bitmarkInfo, err := bitmark.Get(bitmarkID, false)
			if err != nil {
				log.Println(err)
			}

			bitmarkChan <- bitmarkInfo
		}(bitmarkID)
	}

	wg.Wait()

	filterredBitmarkID := make([]string, 0)
	for {
		if len(bitmarkChan) == 0 {
			break
		}

		b, ok := <-bitmarkChan
		if !ok {
			return bitmarks
		}

		if b == nil {
			return bitmarks
		}

		if b.Status != "settled" {
			filterredBitmarkID = append(filterredBitmarkID, b.Id)
		}
	}

	return filterredBitmarkID
}

func WaitForBitmarkConfirmations(bitmarks []string) {
	fmt.Println("Waiting for bitmarks's confirmations")
	filterredBitmarkIDs := bitmarks
	for {
		filterredBitmarkIDs := filterUnconfirmedBitmarks(filterredBitmarkIDs)

		if len(filterredBitmarkIDs) == 0 {
			fmt.Println("Bitmarks are confirmed")
			return
		}

		log.Println("Unconfirmed left:", len(filterredBitmarkIDs))

		time.Sleep(10 * time.Second)
	}
}
