/*
MIT License

Copyright © 2022 William Edwards <shadowapex at gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/shadowblip/steam-shortcut-manager/pkg/chimera"
	"github.com/shadowblip/steam-shortcut-manager/pkg/shortcut"
	"github.com/shadowblip/steam-shortcut-manager/pkg/steam"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a Steam shortcut from your library",
	Long:  `Remove a Steam shortcut from your library`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		format := rootCmd.PersistentFlags().Lookup("output").Value.String()

		// Fetch all users
		users, err := steam.GetUsers()
		if err != nil {
			ExitError(err, format)
		}

		// Check to see if we're fetching for just one user
		onlyForUser := cmd.Flags().Lookup("user").Value.String()

		// Fetch all shortcuts
		for _, user := range users {
			if !steam.HasShortcuts(user) {
				continue
			}
			if onlyForUser != "all" && onlyForUser != user {
				continue
			}

			shortcutsPath, _ := steam.GetShortcutsPath(user)
			shortcuts, err := shortcut.Load(shortcutsPath)
			if err != nil {
				ExitError(err, format)
			}

			// Find the shortcut to remove by name
			shortcutsList := []shortcut.Shortcut{}
			for _, sc := range shortcuts.Shortcuts {
				if sc.AppName == name {
					continue
				}
				shortcutsList = append(shortcutsList, sc)
			}

			// Create a new shortcuts object that we will save
			newShortcuts := &shortcut.Shortcuts{
				Shortcuts: map[string]shortcut.Shortcut{},
			}
			for key, sc := range shortcutsList {
				newShortcuts.Shortcuts[fmt.Sprintf("%v", key)] = sc
			}

			// Write the changes
			err = shortcut.Save(newShortcuts, shortcutsPath)
			if err != nil {
				ExitError(err, format)
			}
		}
	},
}

// removeCmd represents the remove command
var chimeraRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a Chimera shortcut from your library",
	Long:  `Remove a Chimera shortcut from your library`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		format := rootCmd.PersistentFlags().Lookup("output").Value.String()
		DebugPrintln("Using output format:", format)
		if !chimera.HasChimera() {
			ExitError(fmt.Errorf("no chimera config found at %v", chimera.ConfigDir), format)
		}

		// Get the platform flag
		platform := chimeraCmd.PersistentFlags().Lookup("platform").Value.String()

		// Ensure the Chimera shortcuts file exists
		err := chimera.EnsureShortcutsFileExists(platform)
		if err != nil {
			ExitError(err, format)
		}

		// Read from the given shortcuts file
		shortcuts, err := chimera.LoadShortcuts(chimera.GetShortcutsFile(platform))
		if err != nil {
			ExitError(err, format)
		}

		// Find the shortcut to remove by name
		shortcutsList := []*chimera.Shortcut{}
		for _, sc := range shortcuts {
			if sc.Name == name {
				continue
			}
			shortcutsList = append(shortcutsList, sc)
		}

		// Save the shortcuts
		err = chimera.SaveShortcuts(chimera.GetShortcutsFile(platform), shortcutsList)
		if err != nil {
			ExitError(err, format)
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	chimeraCmd.AddCommand(chimeraRemoveCmd)

	removeCmd.Flags().String("user", "all", "Steam user ID to remove the shortcut for")
}
