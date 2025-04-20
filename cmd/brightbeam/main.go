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
		Level: slog.LevelWarn,
	}
)

func cmdLineParse() {
	pflag.StringVarP(&logPath, "log", "l", "", "path to log file. Default is stderr")
	pflag.StringVarP(&treesPath, "trees", "t", "dublin-trees.json", "path to JSON file with group of trees (short/tall)")
	pflag.StringVarP(&propertiesPath, "properties", "p", "dublin-property.csv", "path to CSV file with property prices")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enable verbose (debug) logging")
	pflag.Parse()
}

func initLog() io.Closer {
	if verbose {
		logCfg.Level = slog.LevelDebug
	}
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file %q: %v", logPath, err)
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &logCfg)))
		return f
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &logCfg)))
	return nil
}

func open(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func main() {
	cmdLineParse()
	if l := initLog(); l != nil {
		defer l.Close()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	propertiesSource, err := open(propertiesPath)
	if err != nil {
		slog.Error("CSV open", "error", err)
		if len(os.Args) < 4 {
			pflag.Usage()
		}
		os.Exit(1)
	}
	defer propertiesSource.Close()

	cvsStream, err := streams.NewCsvStream(propertiesSource)
	if err != nil {
		slog.Error("create CSV stream", "error", err)
		os.Exit(2)
	}

	parser, err := csvparser.NewPriceParser(cvsStream, csvparser.WithColNames("Street Name", "Price"))
	if err != nil {
		slog.ErrorContext(ctx, "create price parser", "error", err)
		os.Exit(3)
	}

	jsonSource, err := open(treesPath)
	if err != nil {
		slog.ErrorContext(ctx, "JSON open", "error", err)
		if len(os.Args) < 4 {
			pflag.Usage()
		}

		os.Exit(4)
	}
	defer jsonSource.Close()

	jsonStream := streams.NewJsonStream(jsonSource)
	grouper, groups := groupify.NewTreesGrouper(jsonStream)
	calculator := aggregator.NewAvgPriceBy(groups)

	go func() {
		if err := grouper.GroupStreets(ctx, groups); err != nil {
			slog.ErrorContext(ctx, "group streets", "error", err)
		}
	}()

	prices := make(chan attr.StreetAttribute, 10000)
	go func() {
		if err := parser.ParseAttributes(ctx, prices); err != nil {
			slog.ErrorContext(ctx, "Error parsing prices", "error", err)
		}
	}()

	if result, err := calculator.Process(ctx, prices); err != nil {
		slog.ErrorContext(ctx, "Error processing prices", "error", err)
	} else {
		// Simulate JSON output
		fmt.Println("[")
		for i, g := range result {
			comma := ","
			if i == len(result)-1 {
				comma = ""
			}
			fmt.Printf("  {\"group\":\"%s\",\"average\":\"%s\"}%s\n",
				g.GroupKey(),
				g.AverageValue(),
				comma,
			)
		}
		fmt.Println("]")
	}
}
