// Package steam - artwork application support
package steam

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
)

// AssetType represents the asset types for Steam's SetCustomArtworkForApp API
type AssetType int

const (
	AssetTypeGridPortrait  AssetType = 0 // Capsule 600x900 (grid_p)
	AssetTypeHero          AssetType = 1 // Hero 1920x620
	AssetTypeLogo          AssetType = 2 // Logo
	AssetTypeGridLandscape AssetType = 3 // Wide Capsule 920x430 (grid_l)
	AssetTypeIcon          AssetType = 4 // Icon
)

// ArtworkConfig holds artwork URLs to apply
type ArtworkConfig struct {
	GridPortrait  string // 600x900 portrait grid
	GridLandscape string // 920x430 landscape grid
	HeroImage     string // 1920x620 hero banner
	LogoImage     string // Logo with transparency
	IconImage     string // Square icon
}

// SetArtwork applies artwork for a Steam shortcut.
// Works both locally and remotely (if RemoteClient is set).
// Tries Steam's CEF API first (supports animated WebP/GIF), then falls back
// to the filesystem method if the API is unavailable.
func SetArtwork(appID uint64, artwork *ArtworkConfig) error {
	if artwork == nil {
		return nil
	}

	// Check if aiohttp is available for Steam CEF API method
	canUseSteamAPI := checkAiohttpAvailable()

	// Get grid path for filesystem fallback
	gridPath, err := getGridPath()
	if err != nil {
		return fmt.Errorf("failed to get grid path: %w", err)
	}

	// Helper to apply single artwork with fallback
	applyOne := func(url, baseName string, assetType AssetType) {
		if url == "" {
			return
		}

		success := false
		if canUseSteamAPI {
			if err := SetArtworkViaCEF(appID, url, assetType); err != nil {
				fmt.Printf("[WARNING] Steam CEF API failed for %s: %v\n", baseName, err)
			} else {
				success = true
			}
		}

		if !success {
			// Filesystem fallback
			mkdirAll(gridPath)
			if err := uploadArtworkToGrid(url, gridPath, baseName); err != nil {
				fmt.Printf("[ERROR] Failed to upload %s: %v\n", baseName, err)
			}
		}
	}

	if !canUseSteamAPI {
		fmt.Println("[INFO] Using filesystem method for artwork (static images only)")
		fmt.Println("[INFO] To enable animated WebP/GIF, install: pip install --user aiohttp")
	}

	// Apply all artwork types
	applyOne(artwork.GridPortrait, fmt.Sprintf("%dp", appID), AssetTypeGridPortrait)
	applyOne(artwork.GridLandscape, fmt.Sprintf("%d", appID), AssetTypeGridLandscape)
	applyOne(artwork.HeroImage, fmt.Sprintf("%d_hero", appID), AssetTypeHero)
	applyOne(artwork.LogoImage, fmt.Sprintf("%d_logo", appID), AssetTypeLogo)

	// Icon only via filesystem (Steam API icon handling differs)
	if artwork.IconImage != "" {
		mkdirAll(gridPath)
		if err := uploadArtworkToGrid(artwork.IconImage, gridPath, fmt.Sprintf("%d_icon", appID)); err != nil {
			fmt.Printf("[ERROR] Failed to upload icon: %v\n", err)
		}
	}

	return nil
}

// SetArtworkViaCEF applies artwork using Steam's internal CEF debugger API.
// This method supports animated WebP/GIF images unlike the filesystem method.
// Requires aiohttp Python module (works locally or remotely).
func SetArtworkViaCEF(appID uint64, imageURL string, assetType AssetType) error {
	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download artwork: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read artwork data: %w", err)
	}

	// Write image to temp file
	imagePath := "/tmp/steam_artwork_temp.bin"
	if err := writeFile(imagePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp image: %w", err)
	}

	// Python script that connects to Steam's CEF debugger and calls SetCustomArtworkForApp
	pythonScript := fmt.Sprintf(`
import json
import asyncio
import base64
import aiohttp

async def set_artwork():
    # Read image from temp file
    with open('%s', 'rb') as f:
        image_data = base64.b64encode(f.read()).decode('ascii')

    # Get Steam CEF tabs
    async with aiohttp.ClientSession() as session:
        async with session.get('http://localhost:8080/json') as resp:
            tabs = await resp.json()

    # Find SharedJSContext tab (Steam's main JS context)
    tab = None
    for t in tabs:
        title = t.get('title', '')
        if title in ['SharedJSContext', 'SP', 'Steam']:
            tab = t
            break

    if not tab:
        print('ERROR: Steam SharedJSContext tab not found')
        return False

    ws_url = tab['webSocketDebuggerUrl']

    # Connect to WebSocket and execute JS
    async with aiohttp.ClientSession() as session:
        async with session.ws_connect(ws_url) as ws:
            js_code = f'''
                (async () => {{
                    try {{
                        await SteamClient.Apps.SetCustomArtworkForApp(%d, "{image_data}", "png", %d);
                        return "success";
                    }} catch (e) {{
                        return "error: " + e.message;
                    }}
                }})()
            '''

            await ws.send_json({
                "id": 1,
                "method": "Runtime.evaluate",
                "params": {
                    "expression": js_code,
                    "awaitPromise": True,
                    "userGesture": True
                }
            })

            async for msg in ws:
                if msg.type == aiohttp.WSMsgType.TEXT:
                    result = json.loads(msg.data)
                    if result.get('id') == 1:
                        res = result.get('result', {})
                        if 'exceptionDetails' in res:
                            print('ERROR:', res['exceptionDetails'])
                        else:
                            value = res.get('result', {}).get('value', '')
                            if 'error' in str(value).lower():
                                print('ERROR:', value)
                                return False
                        return True
    return False

import sys
success = asyncio.run(set_artwork())
sys.exit(0 if success else 1)
`, imagePath, appID, assetType)

	// Write and execute the Python script
	scriptPath := "/tmp/steam_set_artwork.py"
	if err := writeFile(scriptPath, []byte(pythonScript), 0755); err != nil {
		return fmt.Errorf("failed to write Python script: %w", err)
	}

	output, err := runCommand("python3", scriptPath)

	// Clean up temp files
	removeFile(scriptPath)
	removeFile(imagePath)

	if err != nil {
		return fmt.Errorf("Steam CEF API failed: %w (output: %s)", err, output)
	}

	if strings.Contains(output, "ERROR") {
		return fmt.Errorf("Steam CEF API error: %s", output)
	}

	return nil
}

// Helper functions that work both locally and remotely

func checkAiohttpAvailable() bool {
	output, err := runCommand("python3", "-c", "import aiohttp")
	if err != nil {
		return false
	}
	return !strings.Contains(output, "ModuleNotFoundError") && !strings.Contains(output, "No module")
}

func getGridPath() (string, error) {
	if IsRemote() {
		users, err := GetRemoteUsers()
		if err != nil || len(users) == 0 {
			return "", fmt.Errorf("no Steam users found")
		}
		userDir, _ := GetRemoteUserDir()
		return path.Join(userDir, users[0], "config", "grid"), nil
	}

	// Local mode
	users, err := GetUsers()
	if err != nil || len(users) == 0 {
		return "", fmt.Errorf("no Steam users found")
	}
	userDir, err := GetUserDir()
	if err != nil {
		return "", err
	}
	return path.Join(userDir, users[0], "config", "grid"), nil
}

func mkdirAll(dir string) error {
	if IsRemote() {
		RemoteClient.RunCommand(fmt.Sprintf("mkdir -p '%s'", dir))
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func writeFile(filePath string, data []byte, perm os.FileMode) error {
	if IsRemote() {
		return RemoteClient.WriteFile(filePath, data, perm)
	}
	return os.WriteFile(filePath, data, perm)
}

func removeFile(filePath string) error {
	if IsRemote() {
		RemoteClient.RunCommand(fmt.Sprintf("rm -f '%s'", filePath))
		return nil
	}
	return os.Remove(filePath)
}

func runCommand(name string, args ...string) (string, error) {
	if IsRemote() {
		cmdStr := name
		for _, arg := range args {
			cmdStr += " " + arg
		}
		output, err := RemoteClient.RunCommand(cmdStr + " 2>&1")
		return string(output), err
	}

	// Local execution
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// uploadArtworkToGrid downloads an image and saves it to the Steam grid folder
func uploadArtworkToGrid(url, gridPath, baseName string) error {
	// Download the image
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download artwork: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read artwork data: %w", err)
	}

	// Determine extension from content type or URL
	ext := getExtensionFromResponse(resp, url)

	// Save to grid folder
	destPath := path.Join(gridPath, baseName+ext)
	return writeFile(destPath, data, 0644)
}

// getExtensionFromResponse determines file extension from HTTP response or URL
func getExtensionFromResponse(resp *http.Response, url string) string {
	contentType := resp.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	}

	// Fallback to URL extension
	urlPath := url
	if idx := strings.Index(url, "?"); idx != -1 {
		urlPath = url[:idx]
	}
	urlLower := strings.ToLower(urlPath)

	switch {
	case strings.HasSuffix(urlLower, ".webp"):
		return ".webp"
	case strings.HasSuffix(urlLower, ".png"):
		return ".png"
	case strings.HasSuffix(urlLower, ".jpg"), strings.HasSuffix(urlLower, ".jpeg"):
		return ".jpg"
	case strings.HasSuffix(urlLower, ".gif"):
		return ".gif"
	default:
		return ".png"
	}
}
