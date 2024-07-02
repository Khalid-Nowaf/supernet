package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

type Writer interface {
	Write(super *supernet.Supernet, directory string, cidrKey string, dropKeys []string) error
	IsIpV6(isIPv6 bool) Writer
}

type JsonWriter struct {
	splitIpVersions bool
	IPv6            bool
	Stats           *Stats
}

func (w *JsonWriter) IsIpV6(isIpV6 bool) Writer {
	w.IPv6 = isIpV6
	return w
}

func (w JsonWriter) Write(super *supernet.Supernet, directory string, cidrCol string, dropKeys []string) error {
	// prepare the file name and extension
	filePath := "resolved"
	if w.splitIpVersions {
		if w.IPv6 {
			filePath += "_v6"
		} else {
			filePath += "_v4"
		}
	}
	filePath += ".json"

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	ipvCidrs := [][]*supernet.CidrTrie{}
	if w.splitIpVersions {
		ipvCidrs = [][]*supernet.CidrTrie{super.AllCIDRS(w.IPv6)}
	} else {
		ipvCidrs = [][]*supernet.CidrTrie{super.AllCIDRS(false), super.AllCIDRS(true)}
	}

	fmt.Println("Starting to write resolved CIDRs...")
	if _, err = file.Write([]byte("[")); err != nil {
		return err
	}
	for _, cidrs := range ipvCidrs {

		for i, cidr := range cidrs {
			// update the the CIDR after resolve
			updateAttributes(cidr, cidrCol, dropKeys)

			if i > 0 {
				if _, err = file.Write([]byte(",")); err != nil {
					return err
				}
			}
			if err = encoder.Encode(cidr.Metadata().Attributes); err != nil {
				return err
			}
			w.Stats.Output++
		}
	}
	if _, err = file.Write([]byte("]")); err != nil {
		return err
	}
	fmt.Println("Writing complete.")
	return nil
}

type CsvWriter struct {
	isTSV           bool
	splitIpVersions bool
	IPv6            bool
	Stats           *Stats
}

func (w *CsvWriter) IsIpV6(isIpV6 bool) Writer {
	w.IPv6 = isIpV6
	return w
}

// writeResults writes the results of CIDR resolution to a CSV file.
func (w CsvWriter) Write(super *supernet.Supernet, directory string, cidrCol string, dropKeys []string) error {

	// prepare the file name and extension
	filePath := "resolved"
	separator := ','
	if w.splitIpVersions {
		if w.IPv6 {
			filePath += "_v6"
		} else {
			filePath += "_v4"
		}
	}
	if w.isTSV {
		filePath += ".tsv"
		separator = '\t'
	} else {
		filePath += ".csv"
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	writer.Comma = separator
	defer writer.Flush()

	ipvCidrs := [][]*supernet.CidrTrie{}
	if w.splitIpVersions {
		ipvCidrs = [][]*supernet.CidrTrie{super.AllCIDRS(w.IPv6)}
	} else {
		ipvCidrs = [][]*supernet.CidrTrie{super.AllCIDRS(false), super.AllCIDRS(true)}
	}

	fmt.Println("Starting to write resolved CIDRs...")

	// Optional: Write headers to the CSV file
	headers := []string{}
	for key := range ipvCidrs[0][0].Metadata().Attributes {
		if !contains(dropKeys, key) {
			headers = append(headers, key)
		}
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data to the CSV file
	for _, cidrs := range ipvCidrs {

		for _, cidr := range cidrs {
			// update the the CIDR after resolve
			updateAttributes(cidr, cidrCol, dropKeys)
			record := make([]string, 0, len(cidr.Metadata().Attributes))
			// Ensure the fields are written in the same order as headers
			for _, header := range headers {
				record = append(record, cidr.Metadata().Attributes[header])
			}
			if err := writer.Write(record); err != nil {
				return err
			}
			w.Stats.Output++
		}
	}

	fmt.Println("Writing complete.")
	return nil
}

func updateAttributes(cidr *supernet.CidrTrie, cidrCol string, dropKeys []string) {
	// update the the CIDR after resolve
	cidr.Metadata().Attributes[cidrCol] = supernet.NodeToCidr(cidr)
	// drop the unwanted keys
	for _, keyToDrop := range dropKeys {
		delete(cidr.Metadata().Attributes, keyToDrop)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
