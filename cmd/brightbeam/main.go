package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"

	"propertytreeanalyzer/pkg/aggregator"
	attr "propertytreeanalyzer/pkg/api/attribute"
	"propertytreeanalyzer/pkg/csvparser"
	"propertytreeanalyzer/pkg/groupify"
	"propertytreeanalyzer/pkg/streams"
)

var (
	logPath        string
	treesPath      string
	propertiesPath string
	verbose        bool
	logCfg         slog.HandlerOptions = slog.HandlerOptions{
		Level: slog.LevelError,
	}
)

func cmdLineParse() {
	pflag.StringVarP(&logPath, "log", "l", "", "path to log file. Default is stdout")
	pflag.StringVarP(&treesPath, "trees", "t", "dublin-trees.json", "path to JSON file with group of trees (short/tall)")
	pflag.StringVarP(&propertiesPath, "properties", "p", "dublin-property.csv", "path to CSV file with property prices")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enable verbose (debug) logging")
	pflag.Parse()
}

func open(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open trees JSON file %q: %v", path, err)
	}
	return file, nil
}

func main() {
	cmdLineParse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if verbose {
		logCfg.Level = slog.LevelDebug
	}
	var output = os.Stdout
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file %q: %v", logPath, err)
		}
		defer f.Close()
		output = f
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(output, &logCfg)))

	propertiesSource, err := open(propertiesPath)
	if err != nil {
		log.Fatalf("failed to open properties CSV file %q: %v", propertiesPath, err)
	}
	defer propertiesSource.Close()

	cvsStream, err := streams.NewCsvStream(propertiesSource)
	if err != nil {
		log.Fatalf("failed to create CSV stream: %v", err)
	}

	parser, err := csvparser.NewPriceParser(cvsStream, csvparser.WithColNames("Street Name", "Price"))
	if err != nil {
		log.Fatalf("failed to create price parser: %v", err)
	}

	jsonSource, err := open(treesPath)
	if err != nil {
		log.Fatalf("failed to open trees JSON file %q: %v", treesPath, err)
	}
	defer jsonSource.Close()

	jsonStream := streams.NewJsonStream(jsonSource)
	grouper, groups := groupify.NewTreesGrouper(jsonStream)
	calculator := aggregator.NewAvgPriceBy(groups)

	go func() {
		if err := grouper.GroupStreets(ctx, groups); err != nil {
			slog.Error("Error grouping streets", "error", err)
		}
	}()

	prices := make(chan attr.StreetAttribute, 10000)
	go func() {
		if err := parser.ParseAttributes(ctx, prices); err != nil {
			slog.Error("Error parsing prices", "error", err)
		}
	}()

	if result, err := calculator.Process(ctx, prices); err != nil {
		slog.Error("Error processing prices", "error", err)
	} else {
		// Simulate JSON output with fmt.Printf
		fmt.Println("[")
		for i, g := range result {
			comma := ","
			if i == len(result)-1 {
				comma = ""
			}
			fmt.Printf("  {\"group\":\"%s\",\"average\":\"%s\"}%s\n",
				g.GroupKey().String(),
				g.AverageValue().String(),
				comma,
			)
		}
		fmt.Println("]")
	}
}
