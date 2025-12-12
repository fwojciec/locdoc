# Go CLI libraries for locdoc: A testability-focused comparison

**Kong emerges as the optimal choice** for locdoc's dependency injection pattern `Main.Run(ctx, args, stdout, stderr)`. It satisfies all critical requirements—testable without `os.Args`, zero global state, mixed positional/flag arguments, and repeatable flags—while maintaining zero external dependencies. Cobra is a close second with equally strong testability but requires the pflag dependency. Libraries like urfave/cli are disqualified due to package-level globals, while stdlib flag cannot handle mixed positional/flag arguments.

## Testing without os.Args: concrete patterns

Each library's testability hinges on whether it can accept an args slice directly and redirect output to custom writers. Here's how each performs:

### Kong: Cleanest testing story

```go
func TestAddCommand(t *testing.T) {
    var cli struct {
        Add struct {
            Name    string   `arg:"" help:"Feed name"`
            URL     string   `arg:"" help:"Feed URL"`
            Preview bool     `help:"Preview mode"`
            Filter  []string `help:"Filter patterns"`
        } `cmd:"" help:"Add a feed"`
    }

    stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
    
    parser, _ := kong.New(&cli,
        kong.Writers(stdout, stderr),  // Redirect all output
        kong.Exit(func(int) { t.Fatal("unexpected exit") }),
    )
    
    // Pass args directly—NO os.Args dependency
    ctx, err := parser.Parse([]string{
        "add", "myfeed", "https://example.com",
        "--preview", "--filter", "*.go", "--filter", "*.md",
    })
    require.NoError(t, err)
    
    assert.Equal(t, "myfeed", cli.Add.Name)
    assert.Equal(t, []string{"*.go", "*.md"}, cli.Add.Filter)
}
```

### Cobra: Equally capable with different API

```go
func executeCommand(root *cobra.Command, args ...string) (string, error) {
    stdout := &bytes.Buffer{}
    root.SetOut(stdout)
    root.SetErr(stdout)
    root.SetArgs(args)  // Bypasses os.Args entirely
    err := root.ExecuteContext(context.Background())
    return stdout.String(), err
}

func TestAddCommand(t *testing.T) {
    cmd := NewAddCmd()  // Factory function returns fresh command
    output, err := executeCommand(cmd, 
        "myfeed", "https://example.com", "--preview", "--filter=*.go")
    require.NoError(t, err)
    assert.Contains(t, output, "Adding myfeed")
}
```

### urfave/cli v3: Works but has gotchas

```go
func TestCommand(t *testing.T) {
    var stdout bytes.Buffer
    cmd := &cli.Command{
        Name:   "myapp",
        Writer: &stdout,  // Captures output
        Action: func(ctx context.Context, cmd *cli.Command) error {
            fmt.Fprintln(cmd.Root().Writer, "output")
            return nil
        },
    }
    // v3 always takes context first
    err := cmd.Run(context.Background(), []string{"myapp", "arg1"})
}
```

### ff/ffcli: Designed for testability

```go
func TestAddCommand(t *testing.T) {
    var preview bool
    fs := flag.NewFlagSet("add", flag.ContinueOnError)
    fs.BoolVar(&preview, "preview", false, "preview mode")
    
    cmd := &ffcli.Command{
        FlagSet: fs,
        Exec: func(ctx context.Context, args []string) error {
            // args contains positional arguments after flag parsing
            return nil
        },
    }
    
    // ParseAndRun takes context and args directly
    err := cmd.ParseAndRun(context.Background(), 
        []string{"--preview", "name", "url"})
}
```

| Library | Test API | Output Capture | Exit Handling |
|---------|----------|----------------|---------------|
| **Kong** | `parser.Parse([]string{})` | `kong.Writers(out, err)` | `kong.Exit(func)` |
| **Cobra** | `cmd.SetArgs([]string{})` | `cmd.SetOut()`, `cmd.SetErr()` | Returns error |
| **urfave/cli** | `cmd.Run(ctx, []string{})` | `cmd.Writer`, `cmd.ErrWriter` | `cli.OsExiter` global |
| **ff** | `cmd.ParseAndRun(ctx, []string{})` | `fs.SetOutput()` | `flag.ContinueOnError` |
| **go-arg** | `parser.Parse([]string{})` | `parser.WriteHelp(w)` | Returns `arg.ErrHelp` |
| **stdlib flag** | `fs.Parse([]string{})` | `fs.SetOutput()` | `flag.ContinueOnError` |

## Global state compatibility with gochecknoglobals

This requirement immediately disqualifies **urfave/cli**. It uses package-level variables that cannot be avoided:

```go
// urfave/cli package-level globals (cannot be disabled)
var OsExiter = os.Exit
var ErrWriter io.Writer = os.Stderr
var HelpPrinter HelpPrinterFunc = DefaultPrintHelp
var VersionPrinter func(cCtx *Context)
```

### Kong and Cobra: Factory pattern eliminates globals

Both libraries work perfectly with local variables and constructor functions:

```go
// Kong: All state in local struct
func NewCLI() *CLI {
    var cli struct {
        Debug bool   `help:"Enable debug"`
        Add   AddCmd `cmd:"" help:"Add feed"`
    }
    return &cli
}

// Cobra: Constructor returns fresh command
func NewRootCmd() *cobra.Command {
    var verbose bool
    cmd := &cobra.Command{Use: "locdoc"}
    cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
    cmd.AddCommand(NewAddCmd())
    return cmd
}
```

**Global state summary:**

| Library | Requires Globals | gochecknoglobals Compatible |
|---------|------------------|----------------------------|
| Kong | No | ✅ Yes |
| Cobra | No | ✅ Yes |
| urfave/cli | **Yes (OsExiter, ErrWriter, HelpPrinter)** | ❌ No |
| ff | No | ✅ Yes |
| go-arg | No | ✅ Yes |
| stdlib flag | No (with NewFlagSet) | ✅ Yes |
| jessevdk/go-flags | No | ✅ Yes |

## Mixed positional and flag arguments: the critical differentiator

The requirement for `locdoc add <name> <url> --preview --force --filter <pattern>` where flags can appear **anywhere** eliminates several libraries:

### Kong: Full interspersed support

```go
var cli struct {
    Add struct {
        Name    string `arg:"" help:"Feed name"`
        URL     string `arg:"" help:"Feed URL"`
        Preview bool   `help:"Preview mode"`
    } `cmd:""`
}

// ALL of these work identically:
// add myfeed https://example.com --preview
// add --preview myfeed https://example.com
// add myfeed --preview https://example.com
```

### Cobra (via pflag): Interspersed by default

```go
cmd := &cobra.Command{
    Use:  "add <name> <url>",
    Args: cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        // args contains ONLY positionals after flag extraction
        name, url := args[0], args[1]
        return nil
    },
}
// pflag parses flags from anywhere before the "--" terminator
```

### urfave/cli and ff: Flags must come first ⚠️

```bash
# urfave/cli and ff (stdlib flag behavior):
myapp add --preview --force name url    # ✅ Works
myapp add name --preview url --force    # ❌ --preview treated as positional!
myapp add name url --preview --force    # ❌ Flags ignored
```

This is a **disqualifying limitation** for locdoc's UX requirements.

### go-arg: Excellent interspersed support

```go
var args struct {
    Add *struct {
        Name    string `arg:"positional,required"`
        URL     string `arg:"positional,required"`
        Preview bool   `arg:"--preview"`
    } `arg:"subcommand:add"`
}
// Handles mixed positional+flags automatically
```

| Library | Mixed Positional+Flags | Notes |
|---------|------------------------|-------|
| **Kong** | ✅ Full support | Struct tags define positionals |
| **Cobra** | ✅ Full support | pflag interspersed mode default |
| **go-arg** | ✅ Full support | Automatic |
| **jessevdk/go-flags** | ⚠️ Via positional struct | Works with struct layout |
| **urfave/cli** | ❌ Flags first only | POSIX-compliant limitation |
| **ff** | ❌ Flags first only | Inherits stdlib flag behavior |
| **stdlib flag** | ❌ Flags first only | Stops at first non-flag |

## Repeatable flags implementation

All libraries support `--filter` specified multiple times, but with varying syntax:

### Kong: Slice type with separator control

```go
var cli struct {
    Filter []string `short:"f" help:"Filter pattern (repeatable)"`
    // Default: --filter=a --filter=b OR --filter=a,b (comma-separated)
}

// To disable comma splitting:
var cli struct {
    Filter []string `short:"f" sep:"none" help:"Must repeat flag"`
}
```

### Cobra: StringSliceVar vs StringArrayVar

```go
var filters []string

// StringSliceVar: supports both repetition AND CSV
cmd.Flags().StringSliceVar(&filters, "filter", nil, "filter pattern")
// --filter=*.go --filter=*.md → ["*.go", "*.md"]
// --filter="*.go,*.md"        → ["*.go", "*.md"]

// StringArrayVar: repetition only (preserves commas in values)
cmd.Flags().StringArrayVar(&filters, "filter", nil, "filter pattern")
// --filter="a,b" → ["a,b"] (single element)
```

### go-arg: `separate` tag

```go
var args struct {
    Filter []string `arg:"-f,--filter,separate" help:"Filter pattern"`
}
// --filter *.go --filter *.md → ["*.go", "*.md"]
```

### ff and stdlib flag: Custom flag.Value required

```go
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
    *s = append(*s, v)  // Accumulate each occurrence
    return nil
}

fs := flag.NewFlagSet("cmd", flag.ContinueOnError)
var filters stringSlice
fs.Var(&filters, "filter", "filter pattern (repeatable)")
```

## Complete boilerplate comparison

**Requirement**: `add <name> <url> [--preview] [--force] [--filter <pattern>...] [-c/--concurrency N]`

### Kong (recommended)

```go
package main

import (
    "context"
    "fmt"
    "github.com/alecthomas/kong"
)

type AddCmd struct {
    Name        string   `arg:"" help:"Feed name"`
    URL         string   `arg:"" help:"Feed URL"`
    Preview     bool     `short:"p" help:"Preview without saving"`
    Force       bool     `short:"f" help:"Force overwrite"`
    Filter      []string `short:"F" help:"Filter patterns (repeatable)"`
    Concurrency int      `short:"c" default:"4" help:"Concurrency level"`
}

func (a *AddCmd) Run(ctx *AppContext) error {
    fmt.Fprintf(ctx.Stdout, "Adding %s from %s\n", a.Name, a.URL)
    return nil
}

type AppContext struct {
    Ctx    context.Context
    Stdout io.Writer
    Stderr io.Writer
}

type CLI struct {
    Add AddCmd `cmd:"" help:"Add a new feed"`
}

// Main.Run pattern implementation
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
    var cli CLI
    parser, err := kong.New(&cli,
        kong.Writers(stdout, stderr),
        kong.Exit(func(int) {}),
    )
    if err != nil {
        return err
    }
    
    kongCtx, err := parser.Parse(args)
    if err != nil {
        return err
    }
    
    appCtx := &AppContext{Ctx: ctx, Stdout: stdout, Stderr: stderr}
    return kongCtx.Run(appCtx)
}
```

### Cobra

```go
package main

import (
    "context"
    "io"
    "github.com/spf13/cobra"
)

type AddOptions struct {
    Preview     bool
    Force       bool
    Filters     []string
    Concurrency int
}

func NewAddCmd() *cobra.Command {
    opts := &AddOptions{}
    
    cmd := &cobra.Command{
        Use:   "add <name> <url>",
        Short: "Add a new feed",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            name, url := args[0], args[1]
            cmd.Printf("Adding %s from %s\n", name, url)
            return nil
        },
    }
    
    cmd.Flags().BoolVarP(&opts.Preview, "preview", "p", false, "Preview mode")
    cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force overwrite")
    cmd.Flags().StringSliceVar(&opts.Filters, "filter", nil, "Filter patterns")
    cmd.Flags().IntVarP(&opts.Concurrency, "concurrency", "c", 4, "Concurrency")
    
    return cmd
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
    root := NewRootCmd()
    root.SetOut(stdout)
    root.SetErr(stderr)
    root.SetArgs(args)
    return root.ExecuteContext(ctx)
}
```

### ff/ffcli

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "io"
    "github.com/peterbourgon/ff/v3/ffcli"
)

type stringSlice []string
func (s *stringSlice) String() string { return "" }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
    addFS := flag.NewFlagSet("add", flag.ContinueOnError)
    addFS.SetOutput(stderr)
    
    preview := addFS.Bool("preview", false, "Preview mode")
    force := addFS.Bool("force", false, "Force overwrite")
    var filters stringSlice
    addFS.Var(&filters, "filter", "Filter pattern (repeatable)")
    concurrency := addFS.Int("c", 4, "Concurrency")
    
    addCmd := &ffcli.Command{
        Name:       "add",
        ShortUsage: "add [flags] <name> <url>",
        FlagSet:    addFS,
        Exec: func(ctx context.Context, args []string) error {
            if len(args) < 2 {
                return fmt.Errorf("requires <name> and <url>")
            }
            // NOTE: flags must come BEFORE positional args!
            fmt.Fprintf(stdout, "Adding %s from %s\n", args[0], args[1])
            return nil
        },
    }
    
    root := &ffcli.Command{
        Subcommands: []*ffcli.Command{addCmd},
    }
    
    return root.ParseAndRun(ctx, args)
}
```

## Dependency analysis

| Library | Direct Dependencies | Total Transitive | Notable |
|---------|--------------------|--------------------|---------|
| **Kong** | **0** | **0** | Zero external deps |
| **ff** | **0** | **0** | Zero runtime deps (testify for tests) |
| **urfave/cli v3** | **0** | **0** | Zero runtime deps |
| **Cobra** | 1 (spf13/pflag) | 1 | Viper is NOT required (separated to cobra-cli) |
| **go-arg** | 1 (go-scalar) | 1 | Minimal |
| **jessevdk/go-flags** | 1 (golang.org/x/sys) | 1 | Terminal operations |
| **mitchellh/cli** | 2 (complete, color) | ~5 | Archived July 2024 |

**Cobra's dependency story has improved significantly**: Prior to v1.4.0, cobra included the CLI scaffolding tool which pulled in viper and ~40 transitive dependencies. This was split to `cobra-cli`, so cobra itself now only depends on pflag.

## Alternative libraries worth considering

### jessevdk/go-flags: Strong contender

```go
type AddOpts struct {
    Preview     bool     `short:"p" long:"preview"`
    Force       bool     `short:"f" long:"force"`
    Filter      []string `long:"filter"`
    Concurrency int      `short:"c" long:"concurrency" default:"1"`
    Args        struct {
        Name string `positional-arg-name:"name" required:"yes"`
        URL  string `positional-arg-name:"url" required:"yes"`
    } `positional-args:"yes"`
}

func Run(args []string) error {
    var opts AddOpts
    _, err := flags.ParseArgs(&opts, args)  // No os.Args
    return err
}
```

**Pros**: Active maintenance (v1.6.1 June 2024), struct tags, testable
**Cons**: 1 dependency (golang.org/x/sys), positional args require nested struct

### jawher/mow.cli: Unique spec string syntax

```go
app := cli.App("add", "Add a feed")
app.Spec = "NAME URL [--preview] [--force] [--filter...]... [-c]"

name := app.StringArg("NAME", "", "Feed name")
filter := app.StringsOpt("filter", nil, "Filter patterns")

app.Action = func() { /* ... */ }
app.Run(args)  // Accepts []string directly
```

**Pros**: Zero dependencies, powerful spec strings for complex arg relationships
**Cons**: Low maintenance (last release Aug 2020), no context support

### mitchellh/cli: Best for DI but archived

```go
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
    ui := &cli.BasicUi{Writer: stdout, ErrorWriter: stderr}
    c := &cli.CLI{
        Name:    "locdoc",
        Args:    args,  // Direct injection
        Commands: map[string]cli.CommandFactory{
            "add": func() (cli.Command, error) {
                return &AddCommand{Ui: ui, Ctx: ctx}, nil
            },
        },
    }
    code, _ := c.Run()
    return code
}
```

**Pros**: Excellent DI pattern, MockUi for testing, factory pattern natural
**Cons**: **Archived July 2024**, no new development

## Recommendation and migration strategy

### Primary recommendation: Kong

Kong is the optimal choice for locdoc because it:

1. **Matches the `Main.Run(ctx, args, stdout, stderr)` pattern perfectly** with `kong.Writers()` and `parser.Parse(args)`
2. **Has zero global state** — all configuration via local structs
3. **Handles mixed positional/flags** with full interspersed support
4. **Has zero dependencies** — simpler supply chain
5. **Supports dependency injection** via `kong.Bind()` for Run methods
6. **Is actively maintained** with good documentation

### Migration strategy from hand-rolled parsing

```go
// Before: Hand-rolled parsing
func parseArgs(args []string) (*Config, error) {
    cfg := &Config{}
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--preview", "-p":
            cfg.Preview = true
        case "--filter":
            i++
            cfg.Filters = append(cfg.Filters, args[i])
        // ... many more cases
        }
    }
    return cfg, nil
}

// After: Kong struct definition
type AddCmd struct {
    Name    string   `arg:"" help:"Feed name"`
    URL     string   `arg:"" help:"Feed URL"`
    Preview bool     `short:"p" help:"Preview mode"`
    Filter  []string `short:"F" help:"Filters"`
}

func (a *AddCmd) Run(ctx *AppContext) error {
    // Business logic moved here
}
```

**Migration steps:**

1. Define CLI struct matching existing flag structure
2. Create `Run(ctx, args, stdout, stderr)` wrapper using `kong.Writers()`
3. Move command logic to `Run()` methods on command structs
4. Update tests to use `parser.Parse([]string{...})`
5. Remove hand-rolled parsing code

### Second choice: Cobra

If the team prefers Cobra's widespread adoption and ecosystem, it's equally viable:

```go
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
    root := NewRootCmd()
    root.SetOut(stdout)
    root.SetErr(stderr)
    root.SetArgs(args)
    return root.ExecuteContext(ctx)
}
```

**When to choose Cobra over Kong:**
- Team already knows Cobra
- Need extensive ecosystem (cobra-cli generator, viper integration)
- Prefer explicit `Flags().StringVar()` over struct tags

## Hidden complexity and gotchas

### Kong gotchas
- **Run() signature must match bindings exactly** — runtime error if types don't match
- **Positional slice args are greedy** — `[]string` positional consumes all remaining args
- **Two tag syntaxes** — both `help:"text"` and `kong:"help='text'"` work, mixing confuses

### Cobra gotchas
- **Must use cmd.OutOrStdout()** — `fmt.Println` bypasses SetOut() capture
- **Context timeouts don't auto-cancel** — must check `ctx.Done()` manually
- **StringSliceVar parses commas** — use StringArrayVar if values contain commas

### urfave/cli gotchas
- **Package globals are unavoidable** — OsExiter, ErrWriter used internally
- **v3 removed cli.Context entirely** — breaking change from v2
- **Flags before args is mandatory** — cannot be disabled

### ff gotchas
- **Inherits stdlib flag limitations** — no interspersed args without workarounds
- **No built-in repeatable flags** — need custom flag.Value in v3

## Final verdict

For locdoc's requirements, the choice is clear:

| Requirement | Kong | Cobra | urfave/cli | ff |
|-------------|------|-------|------------|-----|
| Testable without os.Args | ✅ | ✅ | ✅ | ✅ |
| No global state | ✅ | ✅ | ❌ | ✅ |
| Context support | ✅ (via Bind) | ✅ | ✅ | ✅ |
| Mixed positional+flags | ✅ | ✅ | ❌ | ❌ |
| Repeatable flags | ✅ | ✅ | ✅ | ⚠️ |
| Zero/minimal deps | ✅ (0) | ⚠️ (1) | ✅ (0) | ✅ (0) |
| **RECOMMENDED** | **✅** | **✅** | ❌ | ❌ |

**Use Kong** for the cleanest fit with locdoc's architecture. Use **Cobra** if team familiarity or ecosystem matters more than minimal dependencies.
