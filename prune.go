package main

import "log"

func (cmd *BceCommand) prune(input *BashInput) {
	// build the list of words from the command CmdLine
	words := BashInputToList(input.CmdLine, BashMaxLineSize)

	cmd.pruneArguments(words)
	cmd.pruneSubCommands(words)
}

func (cmd *BceCommand) pruneSubCommands(words []string) {
	var removeIdx []int
	// prune sibling sub-commands
	for i, subCmd := range cmd.SubCommands {
		// check if its in the word list
		subCmd.IsPresentOnCmdLine = contains(words, subCmd.Name)
		if !subCmd.IsPresentOnCmdLine {
			// try harder
			for _, alias := range subCmd.Aliases {
				if contains(words, alias.Name) {
					subCmd.IsPresentOnCmdLine = true
					break
				}
			}
		}
		if subCmd.IsPresentOnCmdLine {
			// update the parent cmd
			cmd.SubCommands[i] = subCmd
			// remove the sub-command's siblings
			for j, sibling := range cmd.SubCommands {
				if subCmd.Uuid != sibling.Uuid {
					removeIdx = append(removeIdx, j)
				}
			}
		}
	}

	// remove the IDs we've collected
	for i := len(removeIdx) - 1; i >= 0; i-- {
		// replace the deleted element with the last element
		//cmd.SubCommands[i] = cmd.SubCommands[len(cmd.SubCommands)-1]
		//cmd.SubCommands = cmd.SubCommands[:len(cmd.SubCommands)-1]
		idx := removeIdx[i]
		log.Println("Removing sub-cmd:", cmd.SubCommands[idx].Name)
		if (len(cmd.SubCommands) == 1) && (idx == 0) {
			cmd.SubCommands = nil
		} else if idx == len(cmd.SubCommands)-1 {
			cmd.SubCommands = cmd.SubCommands[:idx]
		} else {
			cmd.SubCommands = append(cmd.SubCommands[:idx], cmd.SubCommands[idx+1:]...)
		}
	}

	// recurse over remaining sub-cmds
	removeIdx = nil
	for i, subCmd := range cmd.SubCommands {
		subCmd.pruneArguments(words)
		subCmd.pruneSubCommands(words)

		// if sub-cmd is present and has no children, it has been used and should be removed
		if subCmd.IsPresentOnCmdLine && (len(subCmd.SubCommands) == 0) && (len(subCmd.Args) == 0) {
			removeIdx = append(removeIdx, i)
		} else {
			// update the parent
			cmd.SubCommands[i] = subCmd
		}
	}

	for i := len(removeIdx) - 1; i >= 0; i-- {
		// replace the deleted element with the last element
		//cmd.SubCommands[i] = cmd.SubCommands[len(cmd.SubCommands)-1]
		//cmd.SubCommands = cmd.SubCommands[:len(cmd.SubCommands)-1]
		idx := removeIdx[i]
		log.Println("Removing sub-cmd:", cmd.SubCommands[idx].Name)
		if (len(cmd.SubCommands) == 1) && (idx == 0) {
			cmd.SubCommands = nil
		} else if idx == len(cmd.SubCommands)-1 {
			cmd.SubCommands = cmd.SubCommands[:idx]
		} else {
			cmd.SubCommands = append(cmd.SubCommands[:idx], cmd.SubCommands[idx+1:]...)
		}
	}
}

func (cmd *BceCommand) pruneArguments(words []string) {
	var removeIdx []int
	for i, arg := range cmd.Args {
		// check if arg is in word list
		if contains(words, arg.ShortName) || contains(words, arg.LongName) {
			arg.IsPresentOnCmdLine = true
			// update the parent
			cmd.Args[i] = arg
			// check if arg has options
			shouldRemoveArg := false
			if len(arg.Opts) == 0 {
				shouldRemoveArg = true
			} else {
				// possibly remove the arg, if an option has already been supplied
				shouldRemoveArg = true
				for _, opt := range arg.Opts {
					shouldRemoveArg = contains(words, opt.Name)
					if shouldRemoveArg {
						break
					}
				}
			}
			if shouldRemoveArg {
				removeIdx = append(removeIdx, i)
			}
		}
	}

	for i := len(removeIdx) - 1; i >= 0; i-- {
		// replace the deleted element with the last element
		//cmd.Args[i] = cmd.Args[len(cmd.Args)-1]
		//cmd.Args = cmd.Args[:len(cmd.Args)-1]
		idx := removeIdx[i]
		log.Println("Removing arg:", cmd.Args[idx].LongName)
		if (len(cmd.Args) == 1) && (idx == 0) {
			cmd.Args = nil
		} else if idx == len(cmd.Args)-1 {
			cmd.Args = cmd.Args[:idx]
		} else {
			cmd.Args = append(cmd.Args[:idx], cmd.Args[idx+1:]...)
		}
	}
}

func (cmd *BceCommand) CollectRequiredRecommendations(input *BashInput) []string {
	var results []string

	// if a current argument is selected, its options should be displayed 1st
	arg := cmd.GetCurrentArg(*input.CurrentWord)
	if arg == nil {
		return results
	}

	// if ArgType is NONE, don't expect options
	if arg.ArgType != "NONE" {
		for _, opt := range arg.Opts {
			results = append(results, opt.Name)
		}
	}
	return results
}

func (cmd *BceCommand) CollectOptionalRecommendations(input *BashInput) []string {
	var results []string

	// collect all sub-cmds
	for _, subCmd := range cmd.SubCommands {
		if !subCmd.IsPresentOnCmdLine {
			var recommendation string = subCmd.Name
			if len(subCmd.Aliases) > 0 {
				var shortest string
				for _, alias := range subCmd.Aliases {
					if len(shortest) == 0 {
						shortest = alias.Name
					} else if len(alias.Name) < len(shortest) {
						shortest = alias.Name
					}
				}
				recommendation = recommendation + " (" + shortest + ")"
			}
			results = append(results, recommendation)
		}
		subResults := subCmd.CollectOptionalRecommendations(input)
		results = append(results, subResults...)
	}

	// collect all the Args
	for _, arg := range cmd.Args {
		if !arg.IsPresentOnCmdLine {
			var recommendation string
			if len(arg.LongName) > 0 {
				recommendation = arg.LongName
				if len(arg.ShortName) > 0 {
					recommendation = recommendation + " (" + arg.ShortName + ")"
				}
			} else {
				recommendation = arg.ShortName
			}
			results = append(results, recommendation)
		} else {
			// collect all the options
			for _, opt := range arg.Opts {
				results = append(results, opt.Name)
			}
		}
	}

	return results
}

func (cmd *BceCommand) GetCurrentArg(currentWord string) *BceCommandArg {
	var foundArg *BceCommandArg = nil

	for _, arg := range cmd.Args {
		if arg.IsPresentOnCmdLine {
			if (arg.LongName == currentWord) || (arg.ShortName == currentWord) {
				foundArg = &arg
				break
			}
		}
	}

	if foundArg != nil {
		return foundArg
	}

	// recurse for sub-commands
	for _, subCmd := range cmd.SubCommands {
		foundArg = subCmd.GetCurrentArg(currentWord)
		if foundArg != nil {
			break
		}
	}
	return foundArg
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
