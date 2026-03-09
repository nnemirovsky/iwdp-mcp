package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// storageID builds a StorageID for the given origin and storage type.
func storageID(origin string, isLocal bool) webkit.StorageID {
	return webkit.StorageID{
		SecurityOrigin: origin,
		IsLocalStorage: isLocal,
	}
}

// --- Local Storage ---

// GetLocalStorage returns all key-value pairs from local storage for the given origin.
func GetLocalStorage(ctx context.Context, client *webkit.Client, securityOrigin string) ([]webkit.StorageItem, error) {
	return getDOMStorageItems(ctx, client, securityOrigin, true)
}

// SetLocalStorageItem sets a key-value pair in local storage.
func SetLocalStorageItem(ctx context.Context, client *webkit.Client, securityOrigin, key, value string) error {
	return setDOMStorageItem(ctx, client, securityOrigin, true, key, value)
}

// RemoveLocalStorageItem removes a key from local storage.
func RemoveLocalStorageItem(ctx context.Context, client *webkit.Client, securityOrigin, key string) error {
	return removeDOMStorageItem(ctx, client, securityOrigin, true, key)
}

// ClearLocalStorage clears all local storage for the given origin.
func ClearLocalStorage(ctx context.Context, client *webkit.Client, securityOrigin string) error {
	return clearDOMStorageItems(ctx, client, securityOrigin, true)
}

// --- Session Storage ---

// GetSessionStorage returns all key-value pairs from session storage for the given origin.
func GetSessionStorage(ctx context.Context, client *webkit.Client, securityOrigin string) ([]webkit.StorageItem, error) {
	return getDOMStorageItems(ctx, client, securityOrigin, false)
}

// SetSessionStorageItem sets a key-value pair in session storage.
func SetSessionStorageItem(ctx context.Context, client *webkit.Client, securityOrigin, key, value string) error {
	return setDOMStorageItem(ctx, client, securityOrigin, false, key, value)
}

// RemoveSessionStorageItem removes a key from session storage.
func RemoveSessionStorageItem(ctx context.Context, client *webkit.Client, securityOrigin, key string) error {
	return removeDOMStorageItem(ctx, client, securityOrigin, false, key)
}

// ClearSessionStorage clears all session storage for the given origin.
func ClearSessionStorage(ctx context.Context, client *webkit.Client, securityOrigin string) error {
	return clearDOMStorageItems(ctx, client, securityOrigin, false)
}

// --- DOMStorage helpers ---

func getDOMStorageItems(ctx context.Context, client *webkit.Client, securityOrigin string, isLocal bool) ([]webkit.StorageItem, error) {
	result, err := client.Send(ctx, "DOMStorage.getDOMStorageItems", map[string]interface{}{
		"storageId": storageID(securityOrigin, isLocal),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Entries [][]string `json:"entries"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding storage items: %w", err)
	}

	items := make([]webkit.StorageItem, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		if len(entry) != 2 {
			continue
		}
		items = append(items, webkit.StorageItem{
			Key:   entry[0],
			Value: entry[1],
		})
	}
	return items, nil
}

func setDOMStorageItem(ctx context.Context, client *webkit.Client, securityOrigin string, isLocal bool, key, value string) error {
	_, err := client.Send(ctx, "DOMStorage.setDOMStorageItem", map[string]interface{}{
		"storageId": storageID(securityOrigin, isLocal),
		"key":       key,
		"value":     value,
	})
	return err
}

func removeDOMStorageItem(ctx context.Context, client *webkit.Client, securityOrigin string, isLocal bool, key string) error {
	_, err := client.Send(ctx, "DOMStorage.removeDOMStorageItem", map[string]interface{}{
		"storageId": storageID(securityOrigin, isLocal),
		"key":       key,
	})
	return err
}

func clearDOMStorageItems(ctx context.Context, client *webkit.Client, securityOrigin string, isLocal bool) error {
	_, err := client.Send(ctx, "DOMStorage.clearDOMStorageItems", map[string]interface{}{
		"storageId": storageID(securityOrigin, isLocal),
	})
	return err
}

// --- IndexedDB ---

// ListIndexedDatabases returns all IndexedDB databases for the given origin, including their object stores.
func ListIndexedDatabases(ctx context.Context, client *webkit.Client, securityOrigin string) ([]webkit.DatabaseWithObjectStores, error) {
	result, err := client.Send(ctx, "IndexedDB.requestDatabaseNames", map[string]interface{}{
		"securityOrigin": securityOrigin,
	})
	if err != nil {
		return nil, err
	}

	var namesResp struct {
		DatabaseNames []string `json:"databaseNames"`
	}
	if err := json.Unmarshal(result, &namesResp); err != nil {
		return nil, fmt.Errorf("decoding database names: %w", err)
	}

	databases := make([]webkit.DatabaseWithObjectStores, 0, len(namesResp.DatabaseNames))
	for _, name := range namesResp.DatabaseNames {
		dbResult, err := client.Send(ctx, "IndexedDB.requestDatabase", map[string]interface{}{
			"securityOrigin": securityOrigin,
			"databaseName":   name,
		})
		if err != nil {
			return nil, fmt.Errorf("requesting database %q: %w", name, err)
		}

		var dbResp struct {
			DatabaseWithObjectStores webkit.DatabaseWithObjectStores `json:"databaseWithObjectStores"`
		}
		if err := json.Unmarshal(dbResult, &dbResp); err != nil {
			return nil, fmt.Errorf("decoding database %q: %w", name, err)
		}
		databases = append(databases, dbResp.DatabaseWithObjectStores)
	}

	return databases, nil
}

// GetIndexedDBData retrieves data from an IndexedDB object store.
func GetIndexedDBData(ctx context.Context, client *webkit.Client, securityOrigin, databaseName, objectStoreName string, skipCount, pageSize int) (json.RawMessage, error) {
	result, err := client.Send(ctx, "IndexedDB.requestData", map[string]interface{}{
		"securityOrigin":  securityOrigin,
		"databaseName":    databaseName,
		"objectStoreName": objectStoreName,
		"indexName":       "",
		"skipCount":       skipCount,
		"pageSize":        pageSize,
		"keyRange":        nil,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ClearIndexedDBStore clears all data from an IndexedDB object store.
func ClearIndexedDBStore(ctx context.Context, client *webkit.Client, securityOrigin, databaseName, objectStoreName string) error {
	_, err := client.Send(ctx, "IndexedDB.clearObjectStore", map[string]interface{}{
		"securityOrigin":  securityOrigin,
		"databaseName":    databaseName,
		"objectStoreName": objectStoreName,
	})
	return err
}
