package cwe

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/yarburart/str3k0za-radar/internal/domain"
)

// load CWE research csv dataset to memory as value
func LoadCWEdata(path string) (int16, []domain.CWE, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	reader := newCWEReader(file)
	skipHeader(reader)

	cwes := make([]domain.CWE, 0, 1024)
	var lastmaxID int16

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, fmt.Errorf("read csv: %w", err)
		}

		cwe, id, err := parseCWERecord(record)
		if err != nil {
			continue
		}

		cwes = append(cwes, cwe)
		if id > lastmaxID {
			lastmaxID = id
		}
	}

	return lastmaxID, cwes, nil
}

func newCWEReader(r io.Reader) *csv.Reader {
	csvReader := csv.NewReader(r)
	// mitre datasets contain quotes inside descriptions
	csvReader.LazyQuotes = true
	// prevent allocating new string slices per row
	csvReader.ReuseRecord = true
	// variable number of fields per record, just why mitre...
	csvReader.FieldsPerRecord = -1
	return csvReader
}

func skipHeader(r *csv.Reader) {
	_, _ = r.Read()
}

func parseCWERecord(record []string) (domain.CWE, int16, error) {
	if len(record) < 6 {
		return domain.CWE{}, 0, fmt.Errorf("invalid record length")
	}

	idStr := strings.TrimSpace(record[0])
	idStr = strings.TrimPrefix(idStr, "CWE-")

	id, err := strconv.ParseInt(idStr, 10, 16)
	if err != nil {
		return domain.CWE{}, 0, err
	}

	desc := record[4]
	extDesc := record[5]

	// strings cloned cuz ReuseRecord overwrites the buffer
	var finalDesc string
	if extDesc != "" && extDesc != desc {
		var sb strings.Builder
		sb.Grow(len(desc) + len(extDesc) + 4)
		sb.WriteString(desc)
		sb.WriteString("\n\n")
		sb.WriteString(extDesc)
		finalDesc = sb.String()
	} else {
		finalDesc = strings.Clone(desc)
	}

	return domain.CWE{
		ID:          int16(id),
		Name:        strings.Clone(record[1]),
		Description: finalDesc,
	}, int16(id), nil
}
