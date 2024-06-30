package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/khalid_nowaf/supernet/pkg/supernet"
)

type CIDR struct {
	cidr *net.IPNet
	*supernet.Metadata
}

type Record map[string]string

func parseJson(cmd *ResolveCmd, filepath string, onEachCidr func(cidr *CIDR) error) error {
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

func parseCsv(cmd *ResolveCmd, filepath string, onEachCidr func(cidr *CIDR) error) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV Reader
	reader := csv.NewReader(file)

	// Optionally, configure reader fields if necessary (e.g., reader.Comma = ';')
	// reader.Comma = '	' // default delimiter
	// reader.Comment = '#' // example to ignore lines starting with '#'

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
		fmt.Printf("Key: %s CIDR: %s \nRecord: %v", cmd.CidrKey, record[cmd.CidrKey], record)
		return nil, err
	}
	priorityIndex, founded := record[cmd.PriorityKey]
	if founded {
		prioritiesStr := strings.Split(priorityIndex, cmd.PriorityDel)
		for _, priority := range prioritiesStr {
			i, err := strconv.Atoi(priority)

			if err != nil {
				return nil, fmt.Errorf("can not convert priority to Int for record: %v", record)
			}
			priorities = append(priorities, uint8(i))
		}
	} else {
		// TODO: check if the priority at same length, if not (mm maybe we fill the result with Zeros)
		panic("No priorities values founded, use 0 as default " + cmd.PriorityKey)
		priorities = []uint8{0}
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
