package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

type BceCommandJsonWrapper struct {
	Command BceCommand `json:"command"`
}

func processCliImpl() error {
	var err error

	fHelp := flag.Bool("help", false, "get help")
	fExport := flag.String("export", "", "export command")
	fImport := flag.Bool("import", false, "import")
	fFormat := flag.String("format", "sqlite", "file format")
	fFilename := flag.String("filename", "", "file Name")
	fUrl := flag.String("url", "", "URL")
	flag.Parse()

	if *fHelp {
		showUsage()
		return nil
	}

	if len(*fExport) > 0 {
		// ensure we have a filename and a format
		if (len(*fFormat) == 0) || (len(*fFilename) == 0) {
			return errors.New("export requires values for format and file")
		}
		if *fFormat == "json" {
			err = processExportJson(*fExport, *fFilename)
		} else {
			err = processExportSqlite(*fExport, *fFilename)
		}
	} else if *fImport {
		// ensure we have a filename or url
		if (len(*fFilename) == 0) && (len(*fUrl) == 0) {
			return errors.New("import requires values for either file or url")
		}
		if len(*fFilename) > 0 {
			if *fFormat == "json" {
				err = processImportJsonFile(*fFilename)
			} else {
				err = processImportSqlite(*fFilename)
			}
		} else {
			if *fFormat == "json" {
				err = processImportJsonUrl(*fUrl)
			} else {
				return errors.New("import from url must be json format")
			}
		}
	}

	return err
}

func processImportJsonUrl(url string) error {
	return nil
}

func processImportSqlite(filename string) error {
	return nil
}

func processImportJsonFile(filename string) error {
	return nil
}

func processExportSqlite(commandName string, filename string) error {
	// open the source database
	srcConn, err := DbOpen(DbFilename)
	if err != nil {
		return err
	}
	defer DbClose(srcConn)

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

	// open the destination database
	_, err = os.Stat(filename)
	if err == nil {
		err = os.Remove(filename)
	}
	destConn, err := DbOpen(filename)
	if err != nil {
		return err
	}
	defer DbClose(destConn)

	// create the schema
	err = DbCreateSchema(destConn)
	if err != nil {
		return err
	}

	// explicitly start a transaction, since this will be done automatically (per statement) otherwise
	_, err = destConn.Exec("BEGIN TRANSACTION;")
	if err != nil {
		return err
	}

	// insert the BceCommand (recursively to children)
	err = DbInsertCommand(destConn, *cmd)
	if err != nil {
		return err
	}

	// commit the transaction
	_, err = destConn.Exec("COMMIT;")

	return err
}

func processExportJson(commandName string, filename string) error {
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

	wrapper := BceCommandJsonWrapper{*cmd}
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func showUsage() {

}
