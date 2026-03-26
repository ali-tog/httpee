# httpee

`httpee` is a powerful, Terminal-based UI (TUI) HTTP client inspired by [VS Code's REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client). It brings a robust set of features to test API endpoints directly from your terminal using standard `.http` or `.rest` files.

## Features

- **Interactive TUI**: Navigate through your parsed requests using a beautiful split-pane Bubbletea UI.
- **Auto-Discovery**: Automatically parses `.http` and `.rest` files in your current directory.
- **Fuzzy Search**: Filter your parsed requests on the fly immediately upon launch!
- **History Tracking**: Automatically keeps a log of past requests and responses (accessible via `Ctrl+H`).
- **Variables Support**: Native support for declaring variables (e.g., `@name = value`), resolving environment variables (`.env`), and referencing dynamic variables (`{{$datetime}}`, `{{$timestamp}}`).
- **Interactive Live Editing**: Preview the full anatomy of any request by pressing `?`, and edit the request dynamically in-memory using an interactive multi-line terminal text editor!
- **Configurable Keybindings**: Easily customize all keyboard shortcuts by editing your `~/.httpee/config.json` file.
- **Syntax Highlighting**: Perfectly pretty-printed and syntax-highlighted JSON request/response bodies out-of-the-box using Chroma.
- **Shortcuts & Exports**: Press shortcuts to:
  - Copy responses to your clipboard (`c`)
  - Export requests instantly as `cURL` snippets directly to your clipboard (`e`)
  - Toggle HTTP Headers in response (`h`)

## Installation

### macOS / Linux (via curl)

You can easily install the latest pre-built binary using our automated installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/ali-tog/httpee/main/install.sh | bash
```

### Build from Source

```bash
git clone https://github.com/ali-tog/httpee.git
cd httpee
go build -o httpee main.go
# Optional: Migrate the binary to your system PATH
sudo mv httpee /usr/local/bin/
```

## Usage

Create a file named `requests.http` or `requests.rest` in your working directory. You can optionally separate groups of endpoints using `### [Name]` and use variables:

```http
@baseUrl = https://api.github.com

### Get User Profile
GET {{baseUrl}}/users/octocat

### Post Data
POST https://jsonplaceholder.typicode.com/posts
Content-Type: application/json

{
  "title": "foo",
  "body": "bar",
  "userId": 1,
  "time": "{{$datetime}}"
}
```

Run `httpee` in your terminal!

```bash
# Automatically finds local .http and .rest files
httpee

# Or specify explicit files to parse
httpee my_requests.http another.rest
```

### Keybindings

The default keybindings are listed below. You can override these in your `~/.httpee/config.json` file.

- `Tab` - Switch focus between lists and the Response/Preview panel.
- `↑ / ↓`, `j / k`, `PgUp / PgDown` - Move up and down lists.
- `Enter` - Execute request (when list is focused), or enter Editor mode (when preview is open).
- `?` - Inspect request configuration & open Preview mode.
- `v` - Toggle Variables panel to see resolved variables.
- `Ctrl+H` - Toggle History panel to see past requests.
- `q` or `x` - Close the right panel (Response, Preview, Variables).
- `h` - Toggle headers in the response view.
- `c` - Copy raw response body to clipboard.
- `e` - Export currently hovered request as a `cURL` command to your clipboard.
- `Ctrl+S` - Commit dynamic modifications while inside the Interactive Text Editor.
- `Esc` - Cancel interactive editing.
- `Ctrl+C` - Quit application.

## Acknowledgements

- Heavily inspired by the excellent [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) extension for VS Code.
- Built utilizing [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Chroma](https://github.com/alecthomas/chroma).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
