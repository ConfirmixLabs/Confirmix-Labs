package main

import (
	"fmt"
	"log"
	"time"

	"confirmix/pkg/blockchain"
)

func main() {
	// Blockchain'i başlat
	bc := blockchain.NewBlockchain()

	// Validatör ekle
	validator := "validator1"
	bc.AddValidator(validator, "human_proof_1")

	// Validatörün key pair'ini al
	keyPair, exists := bc.GetKeyPair(validator)
	if !exists {
		log.Fatal("Validator key pair not found")
	}

	// Transaction oluştur
	tx := &blockchain.Transaction{
		From:      validator,
		To:        "user1",
		Value:    100,
		Timestamp: time.Now().Unix(),
	}

	// Transaction'ı imzala
	err := tx.Sign(keyPair.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	// Hesap oluştur
	err = bc.CreateAccount(validator, 1000)
	if err != nil {
		log.Fatal(err)
	}

	// Hesap bakiyesini kontrol et
	balance, err := bc.GetBalance(validator)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s bakiyesi: %.2f\n", validator, balance)

	// Transaction'ı blockchain'e ekle
	err = bc.AddTransaction(tx)
	if err != nil {
		log.Fatal(err)
	}

	// Transaction sonrası bakiyeleri güncelle
	err = bc.UpdateBalances(tx)
	if err != nil {
		log.Fatal(err)
	}

	// Güncellenmiş bakiyeyi kontrol et
	balance, err = bc.GetBalance(validator)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s güncellenmiş bakiyesi: %.2f\n", validator, balance)

	// Yeni blok oluştur
	block := blockchain.NewBlock(uint64(len(bc.Blocks)+1), bc.GetPendingTransactions(), bc.Blocks[len(bc.Blocks)-1].Hash, validator, "human_proof_1")

	// Bloğu imzala
	err = block.Sign(keyPair.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	// Bloğu blockchain'e ekle
	bc.AddBlock(block)

	// Sonuçları göster
	fmt.Printf("Blockchain başarıyla oluşturuldu!\n")
	fmt.Printf("Toplam blok sayısı: %d\n", len(bc.Blocks))
	fmt.Printf("Son blok hash: %x\n", bc.Blocks[len(bc.Blocks)-1].Hash)
	fmt.Printf("Son blok validatör: %s\n", bc.Blocks[len(bc.Blocks)-1].Validator)
	fmt.Printf("Son blok timestamp: %d\n", bc.Blocks[len(bc.Blocks)-1].Timestamp)

	// Yeni cüzdan oluştur
	wallet, err := blockchain.CreateWallet()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Yeni cüzdan oluşturuldu! Adres: %s\n", wallet.Address)

	// Cüzdanın genel anahtarını göster
	fmt.Printf("Cüzdan Genel Anahtarı: %x\n", wallet.KeyPair.PublicKey)
}
 