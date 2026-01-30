package steamgriddb

import (
	"fmt"

	"github.com/shadowblip/steam-shortcut-manager/pkg/steam"
)

// FetchArtworkConfig fetches artwork URLs from SteamGridDB for a given game ID
// and returns them as a steam.ArtworkConfig ready to apply
func (c *Client) FetchArtworkConfig(gameID string) (*steam.ArtworkConfig, error) {
	config := &steam.ArtworkConfig{}

	// Fetch portrait grid (600x900)
	gridsPortrait, err := c.GetGrids(gameID, FilterGridVertical())
	if err == nil && len(gridsPortrait.Data) > 0 {
		config.GridPortrait = gridsPortrait.Data[0].URL
	}

	// Fetch landscape grid (920x430)
	gridsLandscape, err := c.GetGrids(gameID, FilterGridHorizontal())
	if err == nil && len(gridsLandscape.Data) > 0 {
		config.GridLandscape = gridsLandscape.Data[0].URL
	}

	// Fetch hero
	heroes, err := c.GetHeroes(gameID)
	if err == nil && len(heroes.Data) > 0 {
		config.HeroImage = heroes.Data[0].URL
	}

	// Fetch logo
	logos, err := c.GetLogos(gameID)
	if err == nil && len(logos.Data) > 0 {
		config.LogoImage = logos.Data[0].URL
	}

	// Fetch icon
	icons, err := c.GetIcons(gameID)
	if err == nil && len(icons.Data) > 0 {
		config.IconImage = icons.Data[0].URL
	}

	return config, nil
}

// ApplyArtwork fetches artwork from SteamGridDB and applies it to a Steam shortcut
func (c *Client) ApplyArtwork(gameID string, appID uint64) error {
	config, err := c.FetchArtworkConfig(gameID)
	if err != nil {
		return fmt.Errorf("failed to fetch artwork config: %w", err)
	}

	return steam.SetArtwork(appID, config)
}

// SearchAndApplyArtwork searches SteamGridDB for a game by name, then fetches
// and applies artwork to a Steam shortcut
func (c *Client) SearchAndApplyArtwork(gameName string, appID uint64) error {
	// Search for the game
	results, err := c.Search(gameName)
	if err != nil {
		return fmt.Errorf("failed to search for game: %w", err)
	}

	if len(results.Data) == 0 {
		return fmt.Errorf("no games found for '%s'", gameName)
	}

	// Use the first result
	gameID := fmt.Sprintf("%d", results.Data[0].ID)

	return c.ApplyArtwork(gameID, appID)
}
