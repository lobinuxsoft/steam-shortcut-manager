/*
MIT License

Copyright © 2022 William Edwards <shadowapex at gmail.com>
*/
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/shadowblip/steam-shortcut-manager/pkg/chimera"
	"github.com/shadowblip/steam-shortcut-manager/pkg/image/kitty"
	"github.com/shadowblip/steam-shortcut-manager/pkg/shortcut"
	"github.com/shadowblip/steam-shortcut-manager/pkg/steam"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List currently registered Steam shortcuts",
	Long:  `Lists all of the shortcuts registered in Steam`,
	Run: func(cmd *cobra.Command, args []string) {
		format := rootCmd.PersistentFlags().Lookup("output").Value.String()

		// Get users
		users, err := steam.GetUsers()
		if err != nil {
			ExitError(err, format)
		}

		// Fetch all shortcuts
		results := map[string]*shortcut.Shortcuts{}
		for _, user := range users {
			if !steam.HasShortcuts(user) {
				continue
			}

			shortcutsPath, _ := steam.GetShortcutsPath(user)
			shortcuts, err := shortcut.Load(shortcutsPath)
			if err != nil {
				ExitError(err, format)
			}

			// Optionally Filter by app id
			if appId, _ := cmd.Flags().GetString("app-id"); appId != "all" {
				newShortcuts := shortcut.NewShortcuts()
				for _, sc := range shortcuts.Shortcuts {
					idStr := fmt.Sprintf("%v", sc.Appid)
					if idStr != appId {
						continue
					}
					newShortcuts.Add(&sc)
				}
				shortcuts = newShortcuts
			}

			// Discover the image paths for the shortcut
			newShortcuts := shortcut.NewShortcuts()
			for _, sc := range shortcuts.Shortcuts {
				idStr := fmt.Sprintf("%v", sc.Appid)
				images := &shortcut.Images{}
				images.Logo, _ = steam.GetImageLogo(user, idStr)
				images.Portrait, _ = steam.GetImagePortrait(user, idStr)
				images.Landscape, _ = steam.GetImageLandscape(user, idStr)
				images.Hero, _ = steam.GetImageHero(user, idStr)
				sc.Images = images
				newShortcuts.Add(&sc)
			}

			results[user] = newShortcuts
		}

		// Print the output
		switch format {
		case "term":
			for user, shortcuts := range results {
				if shortcuts.Shortcuts == nil || len(shortcuts.Shortcuts) == 0 {
					continue
				}
				fmt.Println("User:", user)
				for _, sc := range shortcuts.Shortcuts {
					fmt.Println("  ", sc.AppName)
					fmt.Println("    AppId:         ", sc.Appid)
					fmt.Println("    Executable:    ", sc.Exe)
					fmt.Println("    Launch Options:", sc.LaunchOptions)
					fmt.Println("    Logo Image:    ", sc.Images.Logo)
					if sc.Images.Logo != "" {
						kitty.Display(sc.Images.Logo)
					}
					fmt.Println("    Portrait Image:", sc.Images.Portrait)
					if sc.Images.Portrait != "" {
						kitty.Display(sc.Images.Portrait)
					}
					fmt.Println("    Landscape Image:", sc.Images.Landscape)
					if sc.Images.Landscape != "" {
						kitty.Display(sc.Images.Landscape)
					}
					fmt.Println("    Hero Image:     ", sc.Images.Hero)
					if sc.Images.Hero != "" {
						kitty.Display(sc.Images.Hero)
					}
					fmt.Println("    Icon Image:     ", sc.Icon)
					if sc.Icon != "" {
						kitty.Display(sc.Icon)
					}
				}
			}
		case "json":
			out, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				ExitError(err, format)
			}
			fmt.Println(string(out))
		default:
			panic("unknown output format: " + format)
		}
	},
}

// chimeraListCmd represents the list command
var chimeraListCmd = &cobra.Command{
	Use:   "list",
	Short: "List currently registered Chimera shortcuts",
	Long:  `Lists all of the shortcuts registered in Chimera`,
	Run: func(cmd *cobra.Command, args []string) {
		format := rootCmd.PersistentFlags().Lookup("output").Value.String()
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

		// Print the output
		switch format {
		case "term":
			for _, sc := range shortcuts {
				fmt.Println(sc.Name)
				fmt.Println("  Executable:", sc.Cmd)
			}
		case "json":
			out, err := json.MarshalIndent(shortcuts, "", "  ")
			if err != nil {
				ExitError(err, format)
			}
			fmt.Println(string(out))
		default:
			panic("unknown output format: " + format)
		}

	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	chimeraCmd.AddCommand(chimeraListCmd)

	listCmd.Flags().StringP("app-id", "i", "all", "Only list the given Steam app ID")
}
