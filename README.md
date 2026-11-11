# goda-cmp-ext

The sources and filters of [goda-cmp.vim](https://github.com/Shougo/goda-cmp.vim),
ports of ddc.vim extensions named as the originals without "_"
("matcher_head" is "matcherhead"), and the UI `none`: Go packages for its
configuration, with no Vim script (they are not Vim plugins: nothing to
install but the module).

Their parameters are the original's, as fields of `Params` with the
original's names in JSON (`:help goda-cmp-params`); the package
documentation of each (`go doc`) has the details.

```go
import (
	"github.com/Shougo/goda-cmp-ext/ext/filter/matcherhead"
	"github.com/Shougo/goda-cmp-ext/ext/source/around"
)

b.AddSource(around.New(around.Params{}))
b.AddFilter(matcherhead.New(matcherhead.Params{}))
```

The UIs with Vim script are in repositories of their own:
[goda-cmp-ui-native.vim](https://github.com/Shougo/goda-cmp-ui-native.vim),
[goda-cmp-ui-pum.vim](https://github.com/Shougo/goda-cmp-ui-pum.vim) and
[goda-cmp-ui-inline.vim](https://github.com/Shougo/goda-cmp-ui-inline.vim).

## Sources

| Name | |
|---|---|
| `around` | words around the cursor (ddc-source-around) |
| `buffer` | words of the buffers of the tab page, and of others of the filetype (ddc-source-buffer) |
| `file` | file names from the current directory, the buffer's, and their project roots (ddc-source-file) |
| `register` | register contents, or their words with ExtractWords (ddc-source-register; registers of several lines are offered by their words only) |
| `line` | whole lines around the cursor (ddc-source-line) |
| `mocword` | the next English word, by mocword (ddc-source-mocword) |
| `input` | input() and the expression register (ddc-source-input) |
| `cmdline` | the command line (ddc-source-cmdline) |
| `cmdlinehistory` | the command-line history (ddc-source-cmdline_history) |
| `shellnative` | the completion of zsh, fish or xonsh, run in the background (ddc-source-shell_native) |
| `shell` | ":!" completion of Vim, for Windows (ddc-source-shell) |
| `shellhistory` | commands of shell history files (ddc-source-shell_history) |
| `vim` | Vim script (ddc-source-vim) |
| `lsp` | language servers (`:help goda-cmp-lsp` of goda-cmp.vim) |

## Filters

| Name | |
|---|---|
| `matcherhead` | the head of the word is what is typed |
| `matcherprefix` | the first characters are what is typed (ddc-filter-matcher_prefix) |
| `matcherlength` | longer than what is typed (ddc-filter-matcher_length) |
| `sorterrank` | by match, length and distance from the cursor (ddc-filter-sorter_rank) |
| `sorterhead` | the head of longer items first, for postFilters (ddc-filter-sorter_head) |
| `sorterlength` | shorter first |
| `sortertext` | by Item.SortText (LSP sortText) |
| `sortercmdlinehistory` | found in the command-line history first |
| `sortershellhistory` | found in a shell history file first |
| `converterremoveoverlap` | removes what the text after the cursor already has (ddc-filter-converter_remove_overlap) |
| `convertertruncateabbr` | shortens long labels |

## UI none

`ext/ui/none` (ddc-ui-none): it shows nothing. The items are still
gathered, for the functions that use them without a popup, such as
`goda_cmp#map#insert_item()`. It has no parameters.

```go
import "github.com/Shougo/goda-cmp-ext/ext/ui/none"

b.AddUI(none.New())
b.PatchGlobal(cmp.Config{UI: "none"})
```

```vim
inoremap <expr> <C-t> goda_cmp#map#insert_item(0)
```
