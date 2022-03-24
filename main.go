package main

import (
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
	"os"
)

const DbFilename = "completion.db"

func main() {
	var result = 0
	var args = os.Args[1:]

	if len(args) <= 1 {
		result = processCompletion()
	} else {
		result = processCli(args)
	}

	os.Exit(result)
}

func processCompletion() int {
	//var command_name string
	//var current_word string
	//var previous_word string

	var sqliteVersion, _, _ = sqlite3.Version()
	fmt.Println("SQLite version:", sqliteVersion)

	conn, err := DbOpen(DbFilename)
	if err != nil {
		log.Fatal(err)
		return 1
	}
	defer DbClose(conn)

	schema_version, err := DbGetSchemaVersion(conn)
	if err != nil {
		log.Fatal(err)
		return 1
	}
	if schema_version == 0 {
		// create the schema
		err = DbCreateSchema(conn)
		if err != nil {
			log.Fatal(err)
			return 1
		}
		schema_version, err = DbGetSchemaVersion(conn)
	}
	if schema_version != DbSchemaVersion {
		log.Fatal("Schema version %d", schema_version, "does not match expected version", DbSchemaVersion)
		return 1
	}

	input, err := CreateCompletionInput()
	if err != nil {
		log.Fatal("Unable to load required env variables: " + err.Error())
		return 1
	}

	pCmdName := GetCommandFromInput(input)
	if pCmdName == nil {
		log.Fatal("No command from input: ", input.line)
		return 1
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
		log.Fatal("Unable to query command. err:", err)
		return 1
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

	return 0
}

func processCli(args []string) int {
	return 0
}

func printCommandTree(cmd BceCommand, level int) {
	// indent
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}

	fmt.Println("command:", cmd.name)
	if len(cmd.aliases) > 0 {
		// indent
		for i := 0; i < level; i++ {
			fmt.Print("  ")
		}
		fmt.Print("  aliases: ")
		for _, alias := range cmd.aliases {
			fmt.Print(alias.name, " ")
		}
		fmt.Println()
	}

	if len(cmd.args) > 0 {
		for _, arg := range cmd.args {
			// indent
			for i := 0; i < level; i++ {
				fmt.Print("  ")
			}
			fmt.Printf("  arg: %s (%s): %s\n", arg.longName, arg.shortName, arg.argType)

			// print opts
			if len(arg.opts) > 0 {
				for _, opt := range arg.opts {
					// indent
					for i := 0; i < level; i++ {
						fmt.Print("  ")
					}
					fmt.Printf("    opt: %s\n", opt.name)
				}
			}
		}
	}

	// print sub-commands
	if len(cmd.subCommands) > 0 {
		for _, subCmd := range cmd.subCommands {
			printCommandTree(subCmd, level+1)
		}
	}
}

func printRecommendations(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}
