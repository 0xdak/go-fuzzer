package models

import (
	"database/sql"
	"fmt"
	"log"
)

var DB *sql.DB

type Fuzz struct {
	Id           int
	Url          string
	WordlistFile string
	OutputFile   string
	Ip           string
	WordCount    int
	Started      int64
	Ended        int64
	Finished     int
	Error        int
}

// Fuzzs returns a slice of all books in the books table.
func GetFuzzs() ([]Fuzz, error) {
	rows, err := DB.Query("SELECT * FROM Fuzz")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fuzzs []Fuzz

	for rows.Next() {
		var fuzz Fuzz

		err := rows.Scan(&fuzz.Id, &fuzz.Url, &fuzz.WordlistFile, &fuzz.OutputFile, &fuzz.Ip, &fuzz.WordCount, &fuzz.Started, &fuzz.Ended, &fuzz.Finished, &fuzz.Error)
		if err != nil {
			return nil, err
		}

		fuzzs = append(fuzzs, fuzz)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return fuzzs, nil
}

func InsertFuzz(fuzz Fuzz) (Fuzz, error) {
	stmt, err := DB.Prepare("INSERT INTO Fuzz(Url, WordlistFile, OutputFile, Ip, WordCount, Started, Ended, Finished, Error) values(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return fuzz, err
	}

	res, err := stmt.Exec(fuzz.Url, fuzz.WordlistFile, fuzz.OutputFile, fuzz.Ip, fuzz.WordCount, fuzz.Started, fuzz.Ended, fuzz.Finished, fuzz.Error)
	if err != nil {
		return fuzz, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fuzz, err
	}
	fuzz.Id = int(id)
	return fuzz, nil
}

func UpdateFuzz(fuzz Fuzz) (Fuzz, error) {
	stmt, err := DB.Prepare("UPDATE Fuzz SET Url=?, WordlistFile=?, OutputFile=?, Ip=?, WordCount=?, Started=?, Ended=?, Finished=?, Error=? WHERE Id=?")
	if err != nil {
		return fuzz, err
	}

	res, err := stmt.Exec(fuzz.Url, fuzz.WordlistFile, fuzz.OutputFile, fuzz.Ip, fuzz.WordCount, fuzz.Started, fuzz.Ended, fuzz.Finished, fuzz.Error, fuzz.Id)
	if err != nil {
		return fuzz, err
	}

	affect, err := res.RowsAffected()
	if err != nil {
		return fuzz, err
	}
	fmt.Printf("%s affect", affect)
	return fuzz, nil
}

func GetFuzz(fuzzId int) (Fuzz, error) {
	var fuzz Fuzz

	rows, err := DB.Query("SELECT * FROM Fuzz WHERE Id=?", fuzzId)
	if err != nil {
		return fuzz, err
	}

	for rows.Next() {
		err := rows.Scan(&fuzz.Id, &fuzz.Url, &fuzz.WordlistFile, &fuzz.OutputFile, &fuzz.Ip, &fuzz.WordCount, &fuzz.Started, &fuzz.Ended, &fuzz.Finished, &fuzz.Error)
		if err != nil {
			return fuzz, err
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	rows.Close() //good habit to close

	return fuzz, nil
}
