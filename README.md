# Unique Code Generator

This is a Go program that generates unique codes, stores them in a PostgreSQL database, and can export the data to an Excel file. This project serves as a demonstration of my capabilities in Golang and database management.

## Features

- Generate unique codes with specified length
- Store generated codes in a PostgreSQL database
- Check the number of generated codes
- Export generated codes to an Excel file

## Requirements

- Go 1.16+
- PostgreSQL
- [excelize](https://github.com/xuri/excelize) Go library for Excel operations
- [sqlx](https://github.com/jmoiron/sqlx) Go library for SQL operations

## Installation

1. Clone the repository
    ```sh
    git clone https://github.com/yourusername/unique-code-generator.git
    cd unique-code-generator
    ```
2. Install dependencies
    ```sh
    go get -u github.com/jmoiron/sqlx
    go get -u github.com/lib/pq
    go get -u github.com/xuri/excelize/v2
    ```
3. Setup PostgreSQL database
    ```sql
    CREATE DATABASE your_database_name;
    \c your_database_name
    CREATE TABLE generate_code (
        id SERIAL PRIMARY KEY,
        code VARCHAR(12) UNIQUE NOT NULL
    );
    ```
4. Run the program
    ```sh
    go run main.go
    ```

## Usage

1. Generate Codes
    - Enter the number of codes to generate
2. Check Number of Generated Codes
    - Displays the total number of codes generated so far
3. Export to Excel
    - Exports all generated codes to an Excel file (`./export/codes.xlsx`)

## Configuration

Database configuration is prompted during runtime. The default values are:
- Host: `localhost`
- Port: `5432`
- User: `postgres`
- Password: (empty)
- Database: `postgres`

## Code Overview

- **main.go**: Entry point of the application.
- **generateCode**: Generates unique codes and stores them in the database.
- **getLenOfCode**: Retrieves the number of generated codes from the database.
- **downloadToExcel**: Exports the generated codes to an Excel file.
- **connectDB**: Connects to the PostgreSQL database.
- **initConfigDatabase**: Initializes database configuration by prompting the user.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [excelize](https://github.com/xuri/excelize)
- [sqlx](https://github.com/jmoiron/sqlx)
- [pq](https://github.com/lib/pq)

