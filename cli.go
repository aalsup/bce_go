package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"os"
)

type BceCommandJsonWrapper struct {
	Command BceCommand `json:"command"`
}

func processCli() error {
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
	// open the source database
	srcConn, err := DbOpen(filename)
	if err != nil {
		return err
	}
	defer DbClose(srcConn)

	// explicitly start a transaction, since this will be done automatically (per statement) otherwise
	_, err = srcConn.Exec("BEGIN TRANSACTION;")
	if err != nil {
		return err
	}

	// open the dest database
	destConn, err := DbOpen(DbFilename)
	if err != nil {
		return err
	}
	defer DbClose(destConn)

	// explicitly start a transaction
	_, err = destConn.Exec("BEGIN TRANSACTION;")
	if err != nil {
		return err
	}

	// get a list of the top-level commands in source database
	cmdNames, err := DbQueryRootCommandNames(srcConn)
	if err != nil {
		return err
	}

	// load each command from src and push to dest
	for _, cmdName := range cmdNames {
		cmd, err := DbQueryCommand(srcConn, cmdName)
		if err != nil {
			return err
		}
		err = DbInsertCommand(destConn, *cmd)
		if err != nil {
			return err
		}
	}

	// commit the transaction
	_, err = destConn.Exec("COMMIT;")
	if err != nil {
		return err
	}

	return nil
}

func processImportJsonFile(filename string) error {
	// read in the file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// load the JSON into a map
	var mapData map[string]interface{}
	json.Unmarshal(data, &mapData)

	// convert the map into data model objects
	cmdData := mapData["command"].(map[string]interface{})
	_, err = createBceCommandFromJson(nil, cmdData)
	if err != nil {
		return err
	}

	return nil
}

func createBceCommandFromJson(parentUuid *string, data map[string]interface{}) (*BceCommand, error) {
	cmdUuid, ok := data["uuid"].(string)
	if !ok {
		cmdUuid = uuid.New().String()
	}
	name, ok := data["name"].(string)
	if !ok {
		return nil, errors.New("command.name is a required JSON attribute")
	}

	// collect the aliases
	var aliases []BceCommandAlias
	jAliases, ok := data["aliases"].([]interface{})
	for _, ijAlias := range jAliases {
		jAlias := ijAlias.(map[string]interface{})
		alias, err := createBceCommandAliasFromJson(cmdUuid, jAlias)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, *alias)
	}

	// collect the args
	var args []BceCommandArg
	jArgs, ok := data["args"].([]interface{})
	for _, ijArg := range jArgs {
		jArg := ijArg.(map[string]interface{})
		arg, err := createBceCommandArgFromJson(cmdUuid, jArg)
		if err != nil {
			return nil, err
		}
		args = append(args, *arg)
	}

	var subCmds []BceCommand
	jSubCmds, ok := data["sub_commands"].([]interface{})
	for _, ijSubCmd := range jSubCmds {
		jSubCmd := ijSubCmd.(map[string]interface{})
		subCmd, err := createBceCommandFromJson(&cmdUuid, jSubCmd)
		if err != nil {
			return nil, err
		}
		subCmds = append(subCmds, *subCmd)
	}

	cmd := BceCommand{Uuid: cmdUuid, Name: name, ParentCmdUuid: parentUuid, Args: args}
	return &cmd, nil
}

func createBceCommandArgFromJson(cmdUuid string, data map[string]interface{}) (*BceCommandArg, error) {
	argUuid, ok := data["uuid"].(string)
	if !ok {
		argUuid = uuid.New().String()
	}
	argType, ok := data["arg_type"].(string)
	if !ok {
		return nil, errors.New("arg.arg_type is a required attribute")
	}
	description, ok := data["description"].(string)
	if !ok {
		return nil, errors.New("arg.description is a required attribute")
	}
	longName, ok := data["long_name"].(string)
	shortName, ok := data["short_name"].(string)

	// collect the opts
	var opts []BceCommandOpt
	jOpts, ok := data["opts"].([]interface{})
	for _, ijOpt := range jOpts {
		jOpt := ijOpt.(map[string]interface{})
		opt, err := createBceCommandOptFromJson(argUuid, jOpt)
		if err != nil {
			return nil, err
		}
		opts = append(opts, *opt)
	}

	arg := BceCommandArg{Uuid: argUuid, CmdUuid: cmdUuid, ArgType: argType, Description: description, LongName: longName, ShortName: shortName, Opts: opts}
	return &arg, nil
}

func createBceCommandOptFromJson(argUuid string, data map[string]interface{}) (*BceCommandOpt, error) {
	optUuid, ok := data["uuid"].(string)
	if !ok {
		optUuid = uuid.New().String()
	}
	name, ok := data["name"].(string)
	if !ok {
		return nil, errors.New("opt.name is a required attribute")
	}
	opt := BceCommandOpt{Uuid: optUuid, ArgUuid: argUuid, Name: name}
	return &opt, nil
}

func createBceCommandAliasFromJson(cmdUuid string, data map[string]interface{}) (*BceCommandAlias, error) {
	aliasUuid, ok := data["uuid"].(string)
	if !ok {
		aliasUuid = uuid.New().String()
	}
	name, ok := data["name"].(string)
	if !ok {
		return nil, errors.New("alias.name is a required JSON attribute")
	}
	alias := BceCommandAlias{Uuid: aliasUuid, Name: name, CmdUuid: cmdUuid}
	return &alias, nil
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
