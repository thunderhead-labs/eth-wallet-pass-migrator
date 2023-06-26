package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/wealdtech/go-ecodec"
	e2wallet "github.com/wealdtech/go-eth2-wallet"
	"github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	"github.com/wealdtech/go-eth2-wallet-nd/v2"
	filesystem "github.com/wealdtech/go-eth2-wallet-store-filesystem"
	wtypes "github.com/wealdtech/go-eth2-wallet-types/v2"

	log "github.com/sirupsen/logrus"

	progressbar "github.com/schollz/progressbar/v3"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "eth-wallet-pass-migrator",
		Usage:   "migrate eth2 wallet from one passphrase to another",
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "passphrase",
				Aliases:  []string{"p"},
				Usage:    "passphrase for the existing wallet",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "new-passphrase",
				Aliases:  []string{"np"},
				Usage:    "passphrase for the new wallet",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "wallet-name",
				Aliases:  []string{"wn"},
				Usage:    "name of the existing wallet",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "new-wallet-name",
				Aliases:  []string{"nwn"},
				Usage:    "name of the new wallet",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "store-location",
				Aliases:  []string{"sl"},
				Usage:    "location of the wallet store",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			return run(c.String("passphrase"), c.String("new-passphrase"), c.String("new-wallet-name"), c.String("wallet-name"), c.String("store-location"))
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(passphrase string, passphrase_ string, newWalletName string, walletName string, storeLocation string) error {
	initPassphrase := []byte(passphrase)
	newPassphrase := []byte(passphrase_)

	store := filesystem.New(filesystem.WithPassphrase(initPassphrase), filesystem.WithLocation(storeLocation))

	err := e2wallet.UseStore(store)
	if err != nil {
		return err
	}

	wallet, err := e2wallet.OpenWallet(walletName)
	if err != nil {
		return err
	}

	newWallet, err := nd.CreateWallet(context.Background(), newWalletName, store, keystorev4.New())
	if err != nil {
		return err
	}
	log.Info("Created wallet: ", newWallet.ID())

	accounts := RetrieveAccounts(store, wallet.ID(), wallet, initPassphrase, storeLocation)
	log.Info("Retrieved accounts: ", len(accounts))

	pb := progressbar.NewOptions(len(accounts), progressbar.OptionSetPredictTime(true), progressbar.OptionShowCount())

	for _, account := range accounts {
		pb.Add(1)
		var res map[string]interface{}
		err = json.Unmarshal(account, &res)
		if err != nil {
			return err
		}

		encryptor := keystorev4.New()

		secret, err := encryptor.Decrypt(res["crypto"].(map[string]interface{}), string(initPassphrase))
		if err != nil {
			return err
		}

		newAccount, err := encryptor.Encrypt(secret, string(newPassphrase))
		if err != nil {
			return err
		}

		out, err := json.Marshal(newAccount)
		if err != nil {
			return err
		}

		err = store.StoreAccount(newWallet.ID(), uuid.New(), out)
		if err != nil {
			return err
		}
	}
	pb.Finish()

	log.Info("Done")

	return nil
}

func walletPath(walletID uuid.UUID, store wtypes.Store, location string) string {
	return filepath.FromSlash(filepath.Join(location, walletID.String()))
}

func accountPath(walletID uuid.UUID, accountID uuid.UUID, store wtypes.Store, location string) string {
	return filepath.FromSlash(filepath.Join(location, walletID.String(), accountID.String()))
}

func decryptIfRequired(data []byte, passphrase []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if len(data) < 16 {
		return nil, errors.New("data must be at least 16 bytes")
	}
	var err error
	if len(passphrase) > 0 {
		data, err = ecodec.Decrypt(data, passphrase)
	}
	return data, err
}

func RetrieveAccounts(store wtypes.Store, walletID uuid.UUID, wallet wtypes.Wallet, passphrase []byte, location string) (accounts [][]byte) {
	files, err := os.ReadDir(walletPath(walletID, store, location))
	if err == nil {
		for _, file := range files {
			if file.Name() == walletID.String() {
				continue
			}
			if file.Name() == "index" {
				continue
			}
			accountID, err := uuid.Parse(file.Name())
			if err != nil {
				continue
			}
			data, err := os.ReadFile(accountPath(walletID, accountID, store, location))
			if err != nil {
				continue
			}
			data, err = decryptIfRequired(data, passphrase)
			if err != nil {
				continue
			}

			accounts = append(accounts, data)
		}
	}
	return
}
