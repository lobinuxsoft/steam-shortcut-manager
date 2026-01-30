package shortcut

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wakeful-cloud/vdf"
)

// Load the given shortcuts file
func Load(file string) (*Shortcuts, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Parse the VDF file
	vdfMap, err := vdf.ReadVdf(bytes)
	if err != nil {
		return nil, err
	}

	// Covert to JSON so we can map it to a struct
	rawJSON, err := json.Marshal(vdfMap)
	if err != nil {
		return nil, err
	}

	// Unmarshal to a struct
	var shortcuts Shortcuts
	err = json.Unmarshal(rawJSON, &shortcuts)
	if err != nil {
		return nil, err
	}

	return &shortcuts, nil
}

// Save the given shortcuts file
func Save(shortcuts *Shortcuts, file string) error {
	// Convert the struct to JSON so we can map it to a VDF map
	rawJSON, err := json.Marshal(shortcuts)
	if err != nil {
		return fmt.Errorf("unable to marshal to JSON: %v", err)
	}

	// Marshal the shortcut into a VDF map
	var vdfMap map[string]interface{}
	err = json.Unmarshal(rawJSON, &vdfMap)
	if err != nil {
		return fmt.Errorf("unable to unmarshal to VDF Map: %v", err)
	}

	// Save the shortcuts
	rawVdf, err := vdf.WriteVdf(ensureVDFMap(vdfMap))
	if err != nil {
		return fmt.Errorf("unable to convert VDF to bytes: %v", err)
	}

	// Write the file
	err = os.WriteFile(file, rawVdf, 0666)
	if err != nil {
		return fmt.Errorf("unable to write VDF file: %v", err)
	}

	return nil
}

// ensureVDFMap ensures the given map is a vdf.Map with correct types
func ensureVDFMap(m map[string]interface{}) vdf.Map {
	var newMap vdf.Map = vdf.Map{}
	for k, v := range m {
		if v == nil {
			// Skip nil values - VDF doesn't support them
			continue
		}
		switch val := v.(type) {
		case int:
			newMap[k] = uint32(val)
		case int64:
			newMap[k] = uint32(val)
		case float64:
			newMap[k] = uint32(val)
		case string:
			newMap[k] = val
		case map[string]interface{}:
			newMap[k] = ensureVDFMap(val)
		// Skip any other types that VDF doesn't support
		}
	}
	return newMap
}
