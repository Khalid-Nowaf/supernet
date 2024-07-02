package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

type CIDR struct {
	cidr *net.IPNet
	*supernet.Metadata
}

type Record map[string]string

type CidrParser interface {
	Parse(cmd *ResolveCmd, filepath string, onEachCidr func(cidr *CIDR) error) error
}

type JsonParser struct{}

func (_ JsonParser) Parse(cmd *ResolveCmd, filepath string, onEachCidr func(cidr *CIDR) error) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a JSON Decoder
	decoder := json.NewDecoder(file)

	// Read opening bracket of the array
	_, err = decoder.Token()
	if err != nil {
		return err
	}

	// Decode each element of the array
	for decoder.More() {
		data := Record{}
		err := decoder.Decode(&data)
		if err != nil {
			return err
		}
		cidr, err := parseCIDR(data, cmd)
		if err != nil {
			return err
		}
		onEachCidr(cidr)
	}

	// Read closing bracket of the array
	_, err = decoder.Token()
	if err != nil {
		return err
	}

	return nil
}

type CsvCidrParser struct{ isTSV bool }

func (p CsvCidrParser) Parse(cmd *ResolveCmd, filePath string, onEachCidr func(cidr *CIDR) error) error {
	// extension := filepath.Ext(filePath)
	// if extension != "csv" && extension != "tsv" {
	// 	return fmt.Errorf("File type %s is not supported, please use one of the following [json,csv,tsv]", extension)
	// }

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV Reader
	reader := csv.NewReader(file)

	if p.isTSV {
		reader.Comma = '\t'
	}

	// Read the header to build the key mapping (assuming first line is the header)
	headers, err := reader.Read()
	if err != nil {
		return err
	}

	// Read each record from the CSV
	for {
		recordData, err := reader.Read()
		if err != nil {
			break // End of file or an error
		}

		record := make(Record)
		for i, value := range recordData {
			record[headers[i]] = value
		}

		cidr, err := parseCIDR(record, cmd)
		if err != nil {
			return err
		}
		err = onEachCidr(cidr)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseCIDR(record Record, cmd *ResolveCmd) (*CIDR, error) {
	isV6 := false

	var priorities []uint8

	_, cidr, err := net.ParseCIDR(record[cmd.CidrKey])
	if err != nil {
		return nil, fmt.Errorf("Can not parse CIDR on Key: %s CIDR: %s \nRecord: %v", cmd.CidrKey, record[cmd.CidrKey], record)
	}

	for _, priorityKey := range cmd.PriorityKeys {
		var value int
		// parse priority value
		value, err = strconv.Atoi(record[priorityKey])
		if err != nil {
			if cmd.FillEmptyPriority {
				value = 0
			} else {
				panic(fmt.Sprintf("Can not parse priority %s in record:%v", priorityKey, record))
			}
		}
		// flip priority
		if cmd.FlipRankPriority {
			value = value * -1
		}

		priorities = append(priorities, uint8(value))
	}

	if cidr.IP.To4() == nil {
		isV6 = true
	}
	return &CIDR{
		cidr: cidr,
		Metadata: &supernet.Metadata{
			IsV6:       isV6,
			Priority:   priorities,
			Attributes: record,
		}}, nil
}
