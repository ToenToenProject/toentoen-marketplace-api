package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/account"
	"toentoen-backend/pkg/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config.InitConfig()
	database.ConnectDB()

	var identifier string
	if len(os.Args) > 1 {
		identifier = strings.TrimSpace(os.Args[1])
	}

	if identifier == "" {
		fmt.Print("Masukkan Identifier Akun (Account ID / Email / Username / No. HP / NIK): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Gagal membaca input: %v", err)
		}
		identifier = strings.TrimSpace(input)
	}

	if identifier == "" {
		log.Fatal("Identifier tidak boleh kosong!")
	}

	repo := account.NewProfileRepository(database.DB)
	svc := account.NewProfileService(repo)

	fmt.Printf("\n[PROSES] Mencari dan menghapus akun dengan identifier: %s ...\n", identifier)
	res, err := svc.ForceDeleteAccount(identifier)
	if err != nil {
		log.Fatalf("\n❌ Gagal menghapus akun: %v\n", err)
	}

	fmt.Println("\n✅ AKUN BERHASIL DIHAPUS SECARA PERMANEN!")
	fmt.Printf(" - Account ID       : %s\n", res.AccountID)
	fmt.Printf(" - Identifier Carian: %s\n", res.Identifier)
	fmt.Printf(" - Email            : %s\n", res.Email)
	fmt.Printf(" - Role             : %s\n", res.Role)
	fmt.Printf(" - Bisnis Dihapus   : %d toko (%v)\n", len(res.DeletedBusinessIDs), res.DeletedBusinessIDs)
	fmt.Printf(" - Produk Dihapus   : %d produk\n", len(res.DeletedProductIDs))
	fmt.Println("\nSemua data terkait (KYC, Bisnis, Produk, Toko, Chat, Wishlist, Bookmark, dll.) telah dibersihkan.")
}
