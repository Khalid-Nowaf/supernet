package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/khalid_nowaf/supernet/pkg/supernet"
)

type ResolveCmd struct {
	Files       []string `arg:"" type:"existingfile" help:"Input file containing CIDRs in CSV or JSON format"`
	CidrKey     string   `help:"Index of the CIDRs in the file" default:"cidr"`
	PriorityKey string   `help:"Index of the CIDRs priorities" default:"priority"`
	PriorityDel string   `help:"Delimiter for priorities in the field" default:" "`
	Report      bool     `help:"Report only conflicted CIDRs"`
}

// Run executes the resolve command.
func (cmd *ResolveCmd) Run(ctx *Context) error {
	fmt.Printf("%v \n", *cmd)

	for _, file := range cmd.Files {
		if err := parseAndInsertCidrs(ctx.super, cmd, file); err != nil {
			return err
		}
		if err := writeCsvResults(ctx.super, ".", cmd.CidrKey); err != nil {

		}
	}

	return nil
}

// parseAndInsertCidrs parses a file and inserts CIDRs into the supernet.
func parseAndInsertCidrs(super *supernet.Supernet, cmd *ResolveCmd, file string) error {
	return parseCsv(cmd, file, func(cidr *CIDR) error {
		result := super.InsertCidr(cidr.cidr, cidr.Metadata)
		fmt.Println(result.String())
		return nil
	})
}

// writeResults writes the results of CIDR resolution to a JSON file.
func writeJsonResults(super *supernet.Supernet, directory string, cidrCol string) error {
	filePath := directory + "/resolved.json"
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	cidrs := super.AllCIDRS(false)

	fmt.Println("Starting to write resolved CIDRs...")
	if _, err = file.Write([]byte("[")); err != nil {
		return err
	}

	for i, cidr := range cidrs {
		cidr.Metadata().Attributes[cidrCol] = supernet.NodeToCidr(cidr)
		if i > 0 {
			if _, err = file.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err = encoder.Encode(cidr.Metadata().Attributes); err != nil {
			return err
		}
	}

	if _, err = file.Write([]byte("]")); err != nil {
		return err
	}
	fmt.Println("Writing complete.")
	return nil
}

// writeResults writes the results of CIDR resolution to a CSV file.
func writeCsvResults(super *supernet.Supernet, directory string, cidrCol string) error {
	filePath := directory + "/resolved.csv"
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	cidrs := super.AllCIDRS(false)

	fmt.Println("Starting to write resolved CIDRs...")

	// Optional: Write headers to the CSV file
	headers := []string{}
	for key := range cidrs[0].Metadata().Attributes {
		headers = append(headers, key)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data to the CSV file
	for _, cidr := range cidrs {
		cidr.Metadata().Attributes[cidrCol] = supernet.NodeToCidr(cidr)
		record := make([]string, 0, len(cidr.Metadata().Attributes))
		// Ensure the fields are written in the same order as headers
		for _, header := range headers {
			record = append(record, cidr.Metadata().Attributes[header])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Println("Writing complete.")
	return nil
}
