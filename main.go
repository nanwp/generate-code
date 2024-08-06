package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/xuri/excelize/v2"
)

const (
	dbDriver = "postgres"
)

func main() {
	cfg := initConfigDatabase()
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to database")
	defer db.Close()

	for {
		var choice int
		fmt.Printf("1.\tgenerate kode \n2.\tcek jumlah data \n3. \tgenerate ke excel\n0.\tuntuk keluar\nPilih: ")
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
			result, err := getLenOfCode(db)
			if err != nil {
				log.Printf("Failed to get length of code: %v", err)
			}

			fmt.Printf("Jumlah Code yang sudah di generate : %v\n", result)
		case 3:
			err := dowonloadToExcel(db)
			if err != nil {
				log.Printf("Failed to download to excel: %v", err)
			}

		case 0:
			fmt.Println("Keluar...")
			return
		default:
			fmt.Println("Pilihan tidak valid, silakan coba lagi.")
		}
	}
}

func getLenOfCode(db *sqlx.DB) (int, error) {
	query := `SELECT COUNT(*) FROM generate_code`
	var count int

	err := db.Get(&count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func initConfigDatabase() databaseConfig {
	var cfg databaseConfig

	fmt.Println("Konfigurasi Database")
	fmt.Printf("Masukan host (default: localhost): ")
	_, err := fmt.Scanln(&cfg.host)
	if err != nil {
		cfg.host = "localhost"
	}
	fmt.Printf("Masukan port (default: 5432): ")
	_, err = fmt.Scanln(&cfg.port)
	if err != nil {
		cfg.port = "5432"
	}
	fmt.Printf("Masukan user (default: postgres): ")
	_, err = fmt.Scanln(&cfg.user)
	if err != nil {
		cfg.user = "postgres"
	}
	fmt.Printf("Masukan password (default: ): ")
	_, err = fmt.Scanln(&cfg.pass)
	if err != nil {
		cfg.pass = ""
	}
	fmt.Printf("Masukan database (default: postgres): ")
	_, err = fmt.Scanln(&cfg.name)
	if err != nil {
		cfg.name = "postgres"
	}

	return cfg
}

func generateCode(numCodes int, db *sqlx.DB) error {
	batchSize := 200
	numGoroutines := numCodes / batchSize
	// numGoroutines := 20
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
	pool = "ACDEFGHJKLMNPQRTUVWXYZ234679"
)

type databaseConfig struct {
	user string
	pass string
	name string
	host string
	port string
}

func connectDB(cfg databaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect(dbDriver, "user="+cfg.user+" password="+cfg.pass+" dbname="+cfg.name+" host="+cfg.host+" port="+cfg.port+" sslmode=disable")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(15)
	db.SetConnMaxLifetime(60 * time.Minute)
	db.SetConnMaxIdleTime(30 * time.Minute)

	return db, nil
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

func dowonloadToExcel(db *sqlx.DB) error {
	excelFile := excelize.NewFile()
	sheetPage := 1
	sheetName := fmt.Sprintf("Sheet%d", sheetPage)

	// Set header
	excelFile.SetCellValue(sheetName, "A1", "Code")

	// Get data
	pageSize := 10000
	page := 0

	var mu sync.Mutex
	row := 2
	prefix := "A"

	totalData, err := getLenOfCode(db)
	if err != nil {
		log.Printf("Failed to get length of code: %v\n", err)
		return err
	}

	fmt.Printf("Total data: %d\n", totalData)

	var exportedData int

	for {
		codes, err := getData(db, pageSize, page)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break // No more data to retrieve, break the loop
			}
			log.Printf("Failed to get data: %v\n", err)
			return err
		}

		for _, code := range codes {
			if row > 1000000 {
				sheetPage++
				sheetName = fmt.Sprintf("Sheet%d", sheetPage)
				excelFile.NewSheet(sheetName)
				excelFile.SetCellValue(sheetName, "A1", "Code")
				row = 2
			}
			cell := fmt.Sprintf("%s%d", prefix, row)
			mu.Lock()
			if err := excelFile.SetCellValue(sheetName, cell, code); err != nil {
				log.Printf("Failed to set cell value: %v\n", err)
				return err
			}
			mu.Unlock()
			row++
		}

		exportedData += len(codes)
		fmt.Printf("Exported: %d/%d data\n", exportedData, totalData)

		if len(codes) < pageSize {
			break
		}

		page++
	}

	mu.Lock()
	err = excelFile.SaveAs("./export/codes.xlsx")
	mu.Unlock()
	if err != nil {
		log.Printf("Failed to write to file: %v\n", err)
		return err

	}

	return nil
}

func getData(db *sqlx.DB, pageSize int, page int) ([]string, error) {
	var codes []string
	err := db.Select(&codes, "SELECT code FROM generate_code ORDER BY code LIMIT $1 OFFSET $2", pageSize, pageSize*page)
	if err != nil {
		return nil, err
	}
	return codes, nil
}
