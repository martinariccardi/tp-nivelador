package client

import (
	"fmt"
	"strings"
)

const EXPECTED_CSV_FIELDS = 5

type Bet struct {
	AgencyId  string
	FirstName string
	LastName  string
	Id        string
	Birthdate string
	BetNumber string
}

func parseBetFromCsv(line string, agencyId string) (Bet, error) {
	parts := strings.Split(line, ",")

	if len(parts) != EXPECTED_CSV_FIELDS {
		return Bet{}, fmt.Errorf("Se esperaban 5 campos")
	}

	bet := Bet{
		AgencyId:  agencyId,
		FirstName: parts[0],
		LastName:  parts[1],
		Id:        parts[2],
		Birthdate: parts[3],
		BetNumber: parts[4],
	}

	return bet, nil
}
