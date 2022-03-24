package main

import "log"

func pruneCommand(cmd BceCommand, input BashInput) BceCommand {
	// build the list of words from the command line
	words := BashInputToList(input.line, BashMaxLineSize)

	cmd = pruneArguments(cmd, words)
	cmd = pruneSubCommands(cmd, words)

	return cmd
}

func pruneSubCommands(cmd BceCommand, words []string) BceCommand {
	var removeIdx []int
	// prune sibling sub-commands
	for i, subCmd := range cmd.subCommands {
		// check if its in the word list
		subCmd.isPresentOnCmdLine = contains(words, subCmd.name)
		if !subCmd.isPresentOnCmdLine {
			// try harder
			for _, alias := range subCmd.aliases {
				if contains(words, alias.name) {
					subCmd.isPresentOnCmdLine = true
					break
				}
			}
		}
		if subCmd.isPresentOnCmdLine {
			// update the parent cmd's slice
			cmd.subCommands[i] = subCmd
			// remove the sub-command's siblings
			for j, sibling := range cmd.subCommands {
				if subCmd.uuid != sibling.uuid {
					removeIdx = append(removeIdx, j)
				}
			}
		}
	}

	// remove the IDs we've collected
	for i := len(removeIdx) - 1; i >= 0; i-- {
		// replace the deleted element with the last element
		//cmd.subCommands[i] = cmd.subCommands[len(cmd.subCommands)-1]
		//cmd.subCommands = cmd.subCommands[:len(cmd.subCommands)-1]
		idx := removeIdx[i]
		log.Println("Removing sub-cmd:", cmd.subCommands[idx].name)
		if (len(cmd.subCommands) == 1) && (idx == 0) {
			cmd.subCommands = nil
		} else if idx == len(cmd.subCommands)-1 {
			cmd.subCommands = cmd.subCommands[:idx]
		} else {
			cmd.subCommands = append(cmd.subCommands[:idx], cmd.subCommands[idx+1:]...)
		}
	}

	// recurse over remaining sub-cmds
	removeIdx = nil
	for i, subCmd := range cmd.subCommands {
		subCmd = pruneArguments(subCmd, words)
		subCmd = pruneSubCommands(subCmd, words)

		// if sub-cmd is present and has no children, it has been used and should be removed
		if subCmd.isPresentOnCmdLine && (len(subCmd.subCommands) == 0) && (len(subCmd.args) == 0) {
			removeIdx = append(removeIdx, i)
		} else {
			// update the parent
			cmd.subCommands[i] = subCmd
		}
	}

	for i := len(removeIdx) - 1; i >= 0; i-- {
		// replace the deleted element with the last element
		//cmd.subCommands[i] = cmd.subCommands[len(cmd.subCommands)-1]
		//cmd.subCommands = cmd.subCommands[:len(cmd.subCommands)-1]
		idx := removeIdx[i]
		log.Println("Removing sub-cmd:", cmd.subCommands[idx].name)
		if (len(cmd.subCommands) == 1) && (idx == 0) {
			cmd.subCommands = nil
		} else if idx == len(cmd.subCommands)-1 {
			cmd.subCommands = cmd.subCommands[:idx]
		} else {
			cmd.subCommands = append(cmd.subCommands[:idx], cmd.subCommands[idx+1:]...)
		}
	}
	return cmd
}

func pruneArguments(cmd BceCommand, words []string) BceCommand {
	var removeIdx []int
	for i, arg := range cmd.args {
		// check if arg is in word list
		if contains(words, arg.shortName) || contains(words, arg.longName) {
			arg.isPresentOnCmdLine = true
			// check if arg has options
			shouldRemoveArg := false
			if len(arg.opts) == 0 {
				shouldRemoveArg = true
			} else {
				// possibly remove the arg, if an option has already been supplied
				shouldRemoveArg = true
				for _, opt := range arg.opts {
					shouldRemoveArg = contains(words, opt.name)
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
		//cmd.args[i] = cmd.args[len(cmd.args)-1]
		//cmd.args = cmd.args[:len(cmd.args)-1]
		idx := removeIdx[i]
		log.Println("Removing arg:", cmd.args[idx].longName)
		if (len(cmd.args) == 1) && (idx == 0) {
			cmd.args = nil
		} else if idx == len(cmd.args)-1 {
			cmd.args = cmd.args[:idx]
		} else {
			cmd.args = append(cmd.args[:idx], cmd.args[idx+1:]...)
		}
	}
	return cmd
}

func CollectRequiredRecommendations(cmd BceCommand, currentWord string, previousWord string) []string {
	var results []string

	// if a current argument is selected, its options should be displayed 1st
	arg := GetCurrentArg(cmd, currentWord)
	if arg == nil {
		return results
	}

	// if argType is NONE, don't expect options
	if arg.argType != "NONE" {
		for _, opt := range arg.opts {
			results = append(results, opt.name)
		}
	}
	return results
}

func GetCurrentArg(cmd BceCommand, currentWord string) *BceCommandArg {
	var foundArg *BceCommandArg = nil

	for _, arg := range cmd.args {
		if arg.isPresentOnCmdLine {
			if (arg.longName == currentWord) || (arg.shortName == currentWord) {
				foundArg = &arg
				break
			}
		}
	}

	return foundArg
}

func CollectOptionalRecommendations(cmd BceCommand, currentWord string, previousWord string) []string {
	var results []string

	// collect all sub-cmds
	for _, subCmd := range cmd.subCommands {
		if !subCmd.isPresentOnCmdLine {
			var recommendation string = subCmd.name
			if len(subCmd.aliases) > 0 {
				var shortest string
				for _, alias := range subCmd.aliases {
					if len(shortest) == 0 {
						shortest = alias.name
					} else if len(alias.name) < len(shortest) {
						shortest = alias.name
					}
				}
				recommendation = recommendation + " (" + shortest + ")"
			}
			results = append(results, recommendation)
		}
		subResults := CollectOptionalRecommendations(subCmd, currentWord, previousWord)
		results = append(results, subResults...)
	}

	// collect all the args
	for _, arg := range cmd.args {
		if !arg.isPresentOnCmdLine {
			var recommendation string
			if len(arg.longName) > 0 {
				recommendation = arg.longName
				if len(arg.shortName) > 0 {
					recommendation = recommendation + " (" + arg.shortName + ")"
				}
			} else {
				recommendation = arg.shortName
			}
			results = append(results, recommendation)
		} else {
			// collect all the options
			for _, opt := range arg.opts {
				results = append(results, opt.name)
			}
		}
	}

	return results
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
