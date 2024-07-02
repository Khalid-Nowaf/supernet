package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

type Stats struct {
	Input           int
	Output          int
	Conflicted      int
	StartInsertTime time.Time
	EndInsertTime   time.Time
	StartOutputTime time.Time
	EndOutputTime   time.Time
}

type ResolveCmd struct {
	Files             []string `arg:"" type:"existingfile" help:"Input file containing CIDRs in CSV or JSON format"`
	CidrKey           string   `help:"Key/Colum of the CIDRs in the file" default:"cidr"`
	PriorityKeys      []string `help:"Keys/Columns to be used as CIDRs priorities" default:""`
	FillEmptyPriority bool     `help:"Replace empty/null priority with zero value" default:"true"`
	FlipRankPriority  bool     `help:"Make low value priority mean higher priority" default:"false"`
	Report            bool     `help:"Report only conflicted CIDRs"`

	OutputFormat    string   `enum:"json,csv,tsv" default:"csv" help:"Output file format" default:"csv"`
	DropKeys        []string `help:"Keys/Columns to be dropped" default:""`
	SplitIpVersions bool     `help:"Split the results in to separate files based on the CIDR IP version" default:"false"`
	Stats           Stats    `kong:"-"`
}

// Run executes the resolve command.
func (cmd *ResolveCmd) Run(ctx *Context) error {
	cmd.Stats.StartInsertTime = time.Now()

	// we read each record and insert it in supernet
	for _, file := range cmd.Files {
		if err := parseAndInsertCidrs(ctx.super, cmd, file); err != nil {
			return err
		}
	}
	cmd.Stats.EndInsertTime = time.Now()

	// write back the resolved cidrs to file
	var writer Writer
	switch cmd.OutputFormat {
	case "csv":
		writer = &CsvWriter{splitIpVersions: cmd.SplitIpVersions, Stats: &cmd.Stats}
	case "tsv":
		writer = &CsvWriter{splitIpVersions: cmd.SplitIpVersions, isTSV: true, Stats: &cmd.Stats}
	case "json":
		writer = &JsonWriter{splitIpVersions: cmd.SplitIpVersions, Stats: &cmd.Stats}
	default:
		return fmt.Errorf("--output-format %s is not supported, please uses one of the following: [json,csv,tsv]", cmd.OutputFormat)
	}

	cmd.Stats.StartOutputTime = time.Now()

	var err error
	if cmd.SplitIpVersions {
		err = writer.IsIpV6(false).Write(ctx.super, ".", cmd.CidrKey, cmd.DropKeys)
		err = writer.IsIpV6(true).Write(ctx.super, ".", cmd.CidrKey, cmd.DropKeys)
	} else {

		err = writer.Write(ctx.super, ".", cmd.CidrKey, cmd.DropKeys)
	}
	if err != nil {
		return err
	}
	cmd.Stats.EndOutputTime = time.Now()
	printStats(cmd.Stats)
	return nil
}

// parseAndInsertCidrs parses a file and inserts CIDRs into the supernet.
func parseAndInsertCidrs(super *supernet.Supernet, cmd *ResolveCmd, file string) error {
	var parser CidrParser
	extension := filepath.Ext(file)
	switch extension {
	case ".json":
		parser = &JsonParser{}
	case ".csv":
		parser = &CsvCidrParser{}
	case ".tsv":
		parser = &CsvCidrParser{isTSV: true}
	default:
		{
			return fmt.Errorf("File type %s is not supported, please use one of the following [json,csv,tsv]", extension)
		}
	}

	return parser.Parse(cmd, file, func(cidr *CIDR) error {
		result := super.InsertCidr(cidr.cidr, cidr.Metadata)
		if _, noConflict := result.ConflictType.(supernet.NoConflict); noConflict {
			cmd.Stats.Conflicted++
		}
		cmd.Stats.Input++
		return nil
	})
}

func printStats(stats Stats) {
	fmt.Printf("CIDRs Inserted:\t\t\t\t%d\nCIDRs With Conflicts:\t\t\t%d\nTotal CIDRs After Conflict Resolution:\t%d\n", stats.Input, stats.Conflicted, stats.Output)
	fmt.Printf("Conflict Resolution Duration:\t\t%f Sec\n", stats.EndInsertTime.Sub(stats.StartInsertTime).Seconds())
	fmt.Printf("Writing Results Duration:\t\t%f Sec\n", stats.EndOutputTime.Sub(stats.StartOutputTime).Seconds())
	fmt.Printf("Total Time:\t\t\t\t%f Sec\n", stats.EndOutputTime.Sub(stats.StartInsertTime).Seconds())
}
