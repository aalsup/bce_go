package main

import (
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
	"os"
)

const DBFilename = "completion.db"

func main() {
	var err error
	var args = os.Args[1:]

	if len(args) <= 1 {
		err = processCompletion()
	} else {
		err = processCli()
	}

	if err != nil {
		errLog := log.New(os.Stderr, "", 0)
		errLog.Println(err)
		os.Exit(1)
	}
}

func processCompletion() error {
	var sqliteVersion, _, _ = sqlite3.Version()
	fmt.Println("SQLite version:", sqliteVersion)

	conn, err := DBOpen(DBFilename)
	if err != nil {
		return err
	}
	defer DBClose(conn)

	schemaVersion, err := DBGetSchemaVersion(conn)
	if err != nil {
		return err
	}
	if schemaVersion == 0 {
		// create the schema
		err = DBCreateSchema(conn)
		if err != nil {
			return err
		}
		schemaVersion, err = DBGetSchemaVersion(conn)
	}
	if schemaVersion != DBSchemaVersion {
		err := errors.New("schema version mismatch")
		return err
	}

	input, err := CreateCompletionInput()
	if err != nil {
		return err
	}
	if input.CmdName == nil {
		err := errors.New("no command in input")
		return err
	}

	fmt.Println("input:", input.CmdLine)
	fmt.Println("command:", input.CmdName)
	fmt.Println("current word:", input.CurrentWord)
	fmt.Println("previous word:", input.PreviousWord)

	// search for the command directly (load all descendents)
	cmd, err := DBQueryCommand(conn, *input.CmdName)
	if err != nil {
		return err
	}

	fmt.Println("\nCommand Tree (Database)")
	printCommandTree(cmd, 0)

	// remove non-relevant command data
	*cmd = cmd.prune(*input)

	fmt.Println("\nCommand tree (Pruned)")
	printCommandTree(cmd, 0)

	// build the command recommendations
	var hasRequired = true
	var recommendationList = cmd.CollectRequiredRecommendations(*input)
	if len(recommendationList) == 0 {
		hasRequired = false
		recommendationList = cmd.CollectOptionalRecommendations(*input)
	}

	if hasRequired {
		fmt.Println("\nRecommendations (Required)")
	} else {
		fmt.Println("\nRecommendations (Optional)")
	}

	printRecommendations(recommendationList)

	return nil
}

func printCommandTree(cmd *BceCommand, level int) {
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
			printCommandTree(&subCmd, level+1)
		}
	}
}

func printRecommendations(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}
