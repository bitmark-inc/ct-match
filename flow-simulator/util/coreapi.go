package util

import (
	"time"

	sdk "github.com/bitmark-inc/bitmark-sdk-go"
	"gopkg.in/matryer/try.v1"
)

func TryToSubmitTransfer(bitmarkid, receiver string, sender *sdk.Account, apiClient *sdk.Client) (string, error) {
	var offerID string
	// log.Println("Submit transfer:", bitmarkid, receiver)
	err := try.Do(
		func(attempt int) (bool, error) {
			shouldRetry := attempt < 10
			// Send bitmark to its asset's registrant
			transferOffer, err := apiClient.SignTransferOffer(sender, bitmarkid, receiver, true)
			if err != nil {
				time.Sleep(10 * time.Second)
				return shouldRetry, err
			}

			oid, err := apiClient.SubmitTransferOffer(sender, transferOffer, nil)
			if err != nil {
				time.Sleep(10 * time.Second)
				return shouldRetry, err
			}
			offerID = oid
			return shouldRetry, nil
		},
	)

	return offerID, err
}

func TryToActionTransfer(transferOffer *sdk.TransferOffer, action string, receiver *sdk.Account, apiClient *sdk.Client) (string, error) {
	var tx string

	// log.Println("Action transfer:", action, transferOffer.Id, transferOffer.BitmarkId)

	err := try.Do(
		func(attempt int) (bool, error) {
			shouldRetry := attempt < 10
			counterSign, err := transferOffer.Record.Countersign(receiver)
			if err != nil {
				time.Sleep(10 * time.Second)
				return shouldRetry, err
			}

			t, err := apiClient.CompleteTransferOffer(receiver, transferOffer.Id, action, counterSign.Countersignature)
			if err != nil {
				time.Sleep(10 * time.Second)
				return shouldRetry, err
			}

			tx = t
			return shouldRetry, nil
		},
	)

	return tx, err
}

func TryToTransferOneSignature(sender *sdk.Account, bitmarkID, receiver string, apiClient *sdk.Client) (string, error) {
	var tx string
	// log.Println("transfer with one signature:", bitmarkID, receiver)
	err := try.Do(
		func(attempt int) (bool, error) {
			shouldRetry := attempt < 10
			t, err := apiClient.Transfer(sender, bitmarkID, receiver)
			if err != nil {
				time.Sleep(10 * time.Second)
				return shouldRetry, err
			}
			// log.Println("success transfer with one signature:", tx, bitmarkID, receiver)
			tx = t
			return shouldRetry, nil
		},
	)

	return tx, err
}
