package main

import (
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"os"
)

const DbFilename = "completion.db"

func main() {
	var err error
	var args = os.Args[1:]

	if len(args) <= 1 {
		err = processCompletion()
	} else {
		err = processCli()
	}

	if err != nil {
		os.Exit(1)
	}
}

func processCompletion() error {
	//var command_name string
	//var current_word string
	//var previous_word string

	var sqliteVersion, _, _ = sqlite3.Version()
	fmt.Println("SQLite version:", sqliteVersion)

	conn, err := DbOpen(DbFilename)
	if err != nil {
		return err
	}
	defer DbClose(conn)

	schema_version, err := DbGetSchemaVersion(conn)
	if err != nil {
		return err
	}
	if schema_version == 0 {
		// create the schema
		err = DbCreateSchema(conn)
		if err != nil {
			return err
		}
		schema_version, err = DbGetSchemaVersion(conn)
	}
	if schema_version != DbSchemaVersion {
		err := errors.New("Schema version mismatch")
		return err
	}

	input, err := CreateCompletionInput()
	if err != nil {
		return err
	}

	pCmdName := GetCommandFromInput(input)
	if pCmdName == nil {
		err := errors.New("No command in input")
		return err
	}
	cmdName := *pCmdName

	pCurrentWord := GetCurrentWord(input)
	pPreviousWord := GetPreviousWord(input)

	fmt.Println("input:", input.line)
	fmt.Println("command:", cmdName)
	fmt.Println("current word:", *pCurrentWord)
	fmt.Println("previous word:", *pPreviousWord)

	// explicitly start a transaction, since this will be done automatically (per statement) otherwise
	/*
		_, err = conn.Exec("BEGIN TRANSACTION;")
		if err != nil {
			log.Fatal("Unable to begin transaction. err:", err)
			return 1
		}
	*/

	// search for the command directly (load all descendents)
	cmd, err := DbQueryCommand(conn, cmdName)
	if err != nil {
		return err
	}

	// end the transaction
	/*
		_, err = conn.Exec("COMMIT;")
		if err != nil {
			log.Fatal("Unable to commit transaction. err:", err)
			return 1
		}
	*/

	fmt.Println("\nCommand Tree (Database)")
	printCommandTree(*cmd, 0)

	// remove non-relevant command data
	*cmd = pruneCommand(*cmd, *input)

	fmt.Println("\nCommand tree (Pruned)")
	printCommandTree(*cmd, 0)

	// build the command recommendations
	var hasRequired = true
	var recommendationList = CollectRequiredRecommendations(*cmd, *pCurrentWord, *pPreviousWord)
	if len(recommendationList) == 0 {
		hasRequired = false
		recommendationList = CollectOptionalRecommendations(*cmd, *pCurrentWord, *pPreviousWord)
	}

	if hasRequired {
		fmt.Println("\nRecommendations (Required)")
	} else {
		fmt.Println("\nRecommendations (Optional)")
	}

	printRecommendations(recommendationList)

	return nil
}

func processCli() error {
	err := processCliImpl()
	if err != nil {

	}
	return nil
}

func printCommandTree(cmd BceCommand, level int) {
	// indent
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}

	fmt.Println("command:", cmd.Name)
	if len(cmd.Aliases) > 0 {
		// indent
		for i := 0; i < level; i++ {
			fmt.Print("  ")
		}
		fmt.Print("  Aliases: ")
		for _, alias := range cmd.Aliases {
			fmt.Print(alias.Name, " ")
		}
		fmt.Println()
	}

	if len(cmd.Args) > 0 {
		for _, arg := range cmd.Args {
			// indent
			for i := 0; i < level; i++ {
				fmt.Print("  ")
			}
			fmt.Printf("  arg: %s (%s): %s\n", arg.LongName, arg.ShortName, arg.ArgType)

			// print Opts
			if len(arg.Opts) > 0 {
				for _, opt := range arg.Opts {
					// indent
					for i := 0; i < level; i++ {
						fmt.Print("  ")
					}
					fmt.Printf("    opt: %s\n", opt.Name)
				}
			}
		}
	}

	// print sub-commands
	if len(cmd.SubCommands) > 0 {
		for _, subCmd := range cmd.SubCommands {
			printCommandTree(subCmd, level+1)
		}
	}
}

func printRecommendations(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}
