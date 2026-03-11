package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

func cmdSetCookie(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-cookie <name> <value> <domain> [--path /] [--secure] [--httpOnly] [ws-url]")
		os.Exit(1)
	}
	cookie := webkit.Cookie{
		Name:   args[0],
		Value:  args[1],
		Domain: args[2],
		Path:   "/",
	}
	var wsArgs []string
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				cookie.Path = args[i+1]
				i++
			}
		case "--secure":
			cookie.Secure = true
		case "--httpOnly":
			cookie.HTTPOnly = true
		default:
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetCookie(ctx, client, cookie); err != nil {
		fatal(err)
	}
	fmt.Printf("Cookie %s set\n", cookie.Name)
}

func cmdDeleteCookie(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli delete-cookie <name> <url> [ws-url]")
		os.Exit(1)
	}
	name := args[0]
	cookieURL := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.DeleteCookie(ctx, client, name, cookieURL); err != nil {
		fatal(err)
	}
	fmt.Printf("Cookie %s deleted\n", name)
}

func cmdGetLocalStorage(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-local-storage <origin> [ws-url]")
		os.Exit(1)
	}
	origin := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	items, err := tools.GetLocalStorage(ctx, client, origin)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(out))
}

func cmdSetLocalStorageItem(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-local-storage-item <origin> <key> <value> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[3:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetLocalStorageItem(ctx, client, args[0], args[1], args[2]); err != nil {
		fatal(err)
	}
	fmt.Printf("Set %s = %s\n", args[1], args[2])
}

func cmdRemoveLocalStorageItem(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-local-storage-item <origin> <key> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveLocalStorageItem(ctx, client, args[0], args[1]); err != nil {
		fatal(err)
	}
	fmt.Printf("Removed %s\n", args[1])
}

func cmdClearLocalStorage(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli clear-local-storage <origin> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.ClearLocalStorage(ctx, client, args[0]); err != nil {
		fatal(err)
	}
	fmt.Println("Local storage cleared")
}

func cmdGetSessionStorage(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-session-storage <origin> [ws-url]")
		os.Exit(1)
	}
	origin := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	items, err := tools.GetSessionStorage(ctx, client, origin)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(out))
}

func cmdSetSessionStorageItem(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-session-storage-item <origin> <key> <value> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[3:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetSessionStorageItem(ctx, client, args[0], args[1], args[2]); err != nil {
		fatal(err)
	}
	fmt.Printf("Set %s = %s\n", args[1], args[2])
}

func cmdRemoveSessionStorageItem(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-session-storage-item <origin> <key> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveSessionStorageItem(ctx, client, args[0], args[1]); err != nil {
		fatal(err)
	}
	fmt.Printf("Removed %s\n", args[1])
}

func cmdClearSessionStorage(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli clear-session-storage <origin> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.ClearSessionStorage(ctx, client, args[0]); err != nil {
		fatal(err)
	}
	fmt.Println("Session storage cleared")
}

func cmdListIndexedDatabases(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli list-indexed-databases <origin> [ws-url]")
		os.Exit(1)
	}
	origin := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	dbs, err := tools.ListIndexedDatabases(ctx, client, origin)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(dbs, "", "  ")
	fmt.Println(string(out))
}

func cmdGetIndexedDBData(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-indexed-db-data <origin> <dbName> <storeName> [--skip N] [--count N] [ws-url]")
		os.Exit(1)
	}
	origin := args[0]
	dbName := args[1]
	storeName := args[2]
	skip, count := 0, 10
	var wsArgs []string
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--skip":
			if i+1 < len(args) {
				skip, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--count":
			if i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		default:
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	data, err := tools.GetIndexedDBData(ctx, client, origin, dbName, storeName, skip, count)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(data), "", "  ")
	fmt.Println(string(out))
}

func cmdClearIndexedDBStore(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli clear-indexed-db-store <origin> <dbName> <storeName> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[3:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.ClearIndexedDBStore(ctx, client, args[0], args[1], args[2]); err != nil {
		fatal(err)
	}
	fmt.Println("IndexedDB store cleared")
}
