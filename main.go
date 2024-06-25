package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// func main() {
// 	db, err := connectDB()
// 	if err != nil {
// 		log.Fatalln("Failed to connect to database:", err)
// 	}
// 	defer db.Close()

// 	batchSize := 100
// 	numCodes := 20000000
// 	codes := make([]string, 0, batchSize)

// 	timeStart := time.Now()
// 	defer func() {
// 		timeEnd := time.Now()
// 		log.Printf("Time elapsed: %v\n", timeEnd.Sub(timeStart))
// 	}()

// 	for i := 0; i < numCodes; i++ {
// 		code := generateUniqueCode(12)
// 		codes = append(codes, code)

// 		// Jika sudah mencapai batchSize, lakukan penyimpanan batch
// 		if len(codes) == batchSize || i == numCodes-1 {
// 			err := saveCodesInBatch(db, codes)
// 			if err != nil {
// 				log.Printf("Failed to save batch of codes: %v", err)
// 				// Retry saving the failed batch
// 				i -= len(codes)
// 			}
// 			log.Printf("Generated %d codes\n", i+1)
// 			codes = codes[:0] // Kosongkan slice untuk batch berikutnya
// 		}
// 	}

//		log.Println("Generation complete!")
//	}
func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatalln("Failed to connect to database:", err)
	}
	defer db.Close()

	for {
		var choice int
		fmt.Printf("1.\tgenerate kode, \n2.\texport excel, \n0.\tuntuk keluar\nPilih: ")
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			// Tempatkan logika untuk menghasilkan kode di sini
			fmt.Printf("Masukan Jumlah Kode yang akan dihasilkan: ")
			var numCodes int
			fmt.Scanln(&numCodes)
			err := generateCode(numCodes, db)
			if err != nil {
				log.Printf("Failed to generate codes: %v", err)
			}

		case 2:
			// Tempatkan logika untuk aksi lain di sini
			fmt.Println("Melakukan aksi lain...")
		case 0:
			fmt.Println("Keluar...")
			return
		default:
			fmt.Println("Pilihan tidak valid, silakan coba lagi.")
		}
	}
}

// func generateCode(numCodes int, db *sqlx.DB) error {
// 	batchSize := 100
// 	codes := make([]string, 0, batchSize)
// 	timeStart := time.Now()
// 	defer func() {
// 		timeEnd := time.Now()
// 		log.Printf("Generated %d codes\n", numCodes)
// 		log.Printf("Time elapsed: %v\n", timeEnd.Sub(timeStart))
// 	}()

// 	for i := 0; i < numCodes; i++ {
// 		code := generateUniqueCode(12)
// 		codes = append(codes, code)
// 		if len(codes) == batchSize || i == numCodes-1 {
// 			err := saveCodesInBatch(db, codes)
// 			if err != nil {
// 				if pgErr, ok := err.(*pq.Error); ok && pgErr.Code.Name() == "unique_violation" {
// 					log.Printf("Duplicate code detected. Generating a new one.\n")
// 					i -= len(codes)
// 					codes = codes[:0] // Kosongkan slice untuk batch berikutnya
// 					continue
// 				}

// 				return err
// 			}
// 			log.Printf("Generated %d codes\n", i+1)
// 			codes = codes[:0] // Kosongkan slice untuk batch berikutnya
// 		}
// 	}

// 	log.Println("Generation complete!")
// 	return nil
// }

func generateCode(numCodes int, db *sqlx.DB) error {
	batchSize := 1000
	numGoroutines := numCodes / batchSize
	if numCodes%batchSize != 0 {
		numGoroutines++
	}

	codesChan := make(chan string, batchSize)
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < batchSize && (id*batchSize+j) < numCodes; j++ {
				code := generateUniqueCode(12)
				// Check for duplicate code
				duplicate, err := cekDuplikat(code, db)
				if err != nil {
					log.Printf("Failed to check for duplicate code: %v\n", err)
					return
				}
				if duplicate {
					log.Printf("Duplicate code detected: %s. Generating a new one.\n", code)
					j--
					continue
				}

				codesChan <- code
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(codesChan)
	}()

	codes := make([]string, 0, batchSize)
	for code := range codesChan {
		codes = append(codes, code)
		if len(codes) == batchSize {
			if err := saveCodesInBatch(db, codes); err != nil {
				if pgErr, ok := err.(*pq.Error); ok && pgErr.Code.Name() == "unique_violation" {
					log.Printf("Duplicate code detected. Need a better handling here.\n")
					// Handle duplicate case
				} else {
					return err
				}
			}
			log.Printf("Generated %d codes\n", len(codes))
			codes = codes[:0] // Reset slice for the next batch
		}
	}

	// Handle any remaining codes
	if len(codes) > 0 {
		if err := saveCodesInBatch(db, codes); err != nil {
			return err
		}
	}

	log.Println("Generation complete!")
	return nil
}

func cekDuplikat(code string, db *sqlx.DB) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM generate_code WHERE code = $1", code)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

const (
	pool = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

const (
	dbDriver = "postgres"
	dbUser   = "staging"
	dbPass   = "123PG!"
	dbName   = "ujikom"
	dbHost   = "103.171.182.194"
	dbPort   = "5444"
)

func connectDB() (*sqlx.DB, error) {
	db, err := sqlx.Connect(dbDriver, "user="+dbUser+" password="+dbPass+" dbname="+dbName+" host="+dbHost+" port="+dbPort+" sslmode=disable")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func saveCodeToDB(db *sqlx.DB, code string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	// Use transaction to ensure atomicity
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
		log.Printf("Saved code %s to database\n", code)
	}()

	_, err = tx.Exec("INSERT INTO generate_code (code) VALUES ($1)", code)
	if err != nil {
		// Handle duplicate entry error
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code.Name() == "unique_violation" {
			log.Printf("Duplicate code detected: %s. Generating a new one.\n", code)
			newCode := generateUniqueCode(len(code))
			return saveCodeToDB(db, newCode)
		}
		return err
	}

	return nil
}

func saveCodesInBatch(db *sqlx.DB, codes []string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Use prepared statement for efficiency
	stmt, err := tx.Preparex("INSERT INTO generate_code (code) VALUES ($1)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, code := range codes {
		_, err := stmt.Exec(code)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateUniqueCode(length int) string {
	rand.Seed(time.Now().UnixNano())
	code := make([]byte, length)
	for i := range code {
		code[i] = pool[rand.Intn(len(pool))]
	}
	return string(code)
}
