package protocol

import (
	"fmt"
	"strings"
)

const EXPECTED_CSV_FIELDS = 5

// represents a lottery bet
type Bet struct {
	AgencyId  string
	FirstName string
	LastName  string
	Id        string
	Birthdate string
	BetNumber string
}

// parses a CSV line into a `Bet` value.
// Returns an error when the line does not contain the expected
// number of fields.
func ParseBetFromCsv(line string, agencyId string) (Bet, error) {
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
