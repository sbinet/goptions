/*
package goptions implements a flexible parser for command line options.

Key targets were the support for both long and short flag version, mutually
exclusive flags and verbs. Flags are defined by a (possibly anonymous) struct
and the corresponding tags.

    var options struct {
    	Name string `goptions:"-n, --name"`
    	Force bool `goptions:"-f, --force"`
    	Verbosity int `goptions:"-v, --verbose, accumulate"`
    }

Short flags can be combined (e.g. `-nfv`). Long flags take their value after a
separating space. The equals notation (`--long-flag=value`) is NOT supported
right now.

Every member of the struct, which is supposed to catch a command line value
has to have a "goptions" tag. Multiple short and long flag names can be specified.
Each tag can also list any number of the following options:

    accumulate        - (Only valid for `int`) Counts how of then the flag has been
                        specified in the short version. The long version simply
                        accepts an int.
    obligatory        - Flag must be specified. Otherwise an error will be returned
                        when Parse() is called.
    description='...' - Set the description for this particular flag. Will be
                        used by the HelpFunc.
    mutexgroup='...'  - Sets the name of the MutexGroup. Only one flag of the
                        ones sharing a MutexGroup can be set. Otherwise an error
                        will be returned when Parse() is called.

goptions also has support for verbs. Each verb accepts its own set of flags which
take exactly the same tag format as global options. For an usage example of verbs
see the PrintHelp() example.
*/
package goptions

import (
	"io"
	"os"
	"sync"
	"text/tabwriter"
	"text/template"
)

var (
	globalFlagSet *FlagSet
)

// Parse parses the command-line flags from os.Args[1:].
func Parse(v interface{}) error {
	fs, err := NewFlagSet(os.Args[0], v)
	if err != nil {
		return err
	}
	globalFlagSet = fs

	e := fs.Parse(os.Args[1:])
	if e != nil {
		return e
	}

	return nil
}

// PrintHelp renders the default help to os.Stderr.
func PrintHelp() {
	if globalFlagSet == nil {
		panic("Must call Parse() before PrintHelp()")
	}
	globalFlagSet.PrintHelp(os.Stderr)
}

// Generates a new HelpFunc taking a `text/template.Template`-formatted
// string as an argument. The resulting template will be executed with the FlagSet
// as its data.
func NewTemplatedHelpFunc(tpl string) HelpFunc {
	var once sync.Once
	var t *template.Template
	return func(w io.Writer, fs *FlagSet) {
		once.Do(func() {
			t = template.Must(template.New("helpTemplate").Parse(tpl))
		})
		err := t.Execute(w, fs)
		if err != nil {
			panic(err)
		}
	}
}

// Must is a helper that wraps a call to a function returning (*FlagSet, error)
// and panics if the error is non-nil.
func Must(fs *FlagSet, err error) *FlagSet {
	if err != nil {
		panic(err)
	}
	return fs
}

const (
	_DEFAULT_HELP = `
Usage: {{.Name}} [global options] {{with .Verbs}}<verb> [verb options]{{end}}

Global options:{{range .Flags}}
	{{if len .Short}}-{{index .Short 0}},{{end}}	{{if len .Long}}--{{index .Long 0}}{{end}}	{{.Description}}{{if .Obligatory}} (*){{end}}{{end}}

{{if .Verbs}}Verbs:{{range .Verbs}}
	{{.Name}}:{{range .Flags}}
		{{if len .Short}}-{{index .Short 0}},{{end}}	{{if len .Long}}--{{index .Long 0}}{{end}}	{{.Description}}{{if .Obligatory}} (*){{end}}{{end}}{{end}}{{end}}
`
)

// DefaultHelpFunc is a HelpFunc which renders the default help template and pipes
// the output through a text/tabwriter.Writer before flushing it to the output.
func DefaultHelpFunc(w io.Writer, fs *FlagSet) {
	tw := &tabwriter.Writer{}
	tw.Init(w, 4, 4, 1, ' ', 0)
	NewTemplatedHelpFunc(_DEFAULT_HELP)(tw, fs)
	tw.Flush()
}
