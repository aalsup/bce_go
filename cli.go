package main

import (
	"errors"
	"flag"
	"log"
)

type CliOperation uint8
type CliFormat uint8

const (
	OpNone CliOperation = iota
	OpHelp
	OpImport
	OpExport
)

const (
	FormatSqlite CliFormat = iota
	FormatJson
)

func processCliImpl(args []string) error {
	var err error

	fHelp := flag.Bool("help", false, "get help")
	fExport := flag.Bool("export", false, "export")
	fImport := flag.Bool("import", false, "import")
	fFormat := flag.String("format", "sqlite", "file format")
	fFilename := flag.String("file", "", "file name")
	fUrl := flag.String("url", "", "URL")
	flag.Parse()
	commandName := flag.Arg(0)

	if *fHelp {
		showUsage()
		return nil
	}

	if *fExport {
		// ensure we have a filename and a format
		if (len(*fFormat) == 0) || (len(*fFilename) == 0) {
			return errors.New("export requires values for format and file")
		}
		if *fFormat == "json" {
			err = processExportJson(commandName, *fFilename)
		} else {
			err = processExportSqlite(commandName, *fFilename)
		}
	} else if *fImport {
		// ensure we have a filename or url
		if (len(*fFilename) == 0) && (len(*fUrl) == 0) {
			return errors.New("import requires values for either file or url")
		}
	}

	return err
}

func processExportSqlite(commandName string, filename string) error {
	// open the source database
	srcConn, err := DbOpen(DbFilename)
	if err != nil {
		return err
	}

	// explicitly start a transaction, since this will be done automatically (per statement) otherwise
	_, err = srcConn.Exec("BEGIN TRANSACTION;")
	if err != nil {
		return err
	}

	// load the command hierarchy
	cmd, err := DbQueryCommand(srcConn, commandName)
	if err != nil {
		return err
	}
	log.Println(cmd)

	return err
}

func processExportJson(commandName string, filename string) error {
	var err error

	return err
}

func showUsage() {

}
