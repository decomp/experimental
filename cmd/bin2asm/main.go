// The bin2asm tool disassembles binary executables to x86 assembly (*.exe ->
// *.asm).
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/decomp/exp/bin"
	_ "github.com/decomp/exp/bin/elf" // register ELF decoder
	_ "github.com/decomp/exp/bin/pe"  // register PE decoder
	_ "github.com/decomp/exp/bin/pef" // register PEF decoder
	"github.com/decomp/exp/bin/raw"
	"github.com/decomp/exp/disasm/x86"
	"github.com/kr/pretty"
	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
)

// Loggers.
var (
	// dbg represents a logger with the "bin2asm:" prefix, which logs debug
	// messages to standard error.
	dbg = log.New(os.Stderr, term.YellowBold("bin2asm:")+" ", 0)
	// warn represents a logger with the "warning:" prefix, which logs warning
	// messages to standard error.
	warn = log.New(os.Stderr, term.RedBold("warning:")+" ", 0)
)

func usage() {
	const use = `
Disassemble binary executables to x86 assembly (*.exe -> *.asm).

Usage:

	bin2asm [OPTION]... FILE

Flags:
`
	fmt.Fprint(os.Stderr, use[1:])
	flag.PrintDefaults()
}

func main() {
	// Parse command line arguments.
	var (
		// blockAddr specifies a basic block address to disassemble.
		blockAddr bin.Address
		// TODO: Remove -first flag and firstAddr.
		// firstAddr specifies the first function address to disassemble.
		firstAddr bin.Address
		// funcAddr specifies a function address to disassemble.
		funcAddr bin.Address
		// TODO: Remove -last flag and lastAddr.
		// lastAddr specifies the last function address to disassemble.
		lastAddr bin.Address
		// quiet specifies whether to suppress non-error messages.
		quiet bool
		// rawArch specifies the machine architecture of a raw binary executable.
		rawArch bin.Arch
		// rawEntry specifies the entry point of a raw binary executable.
		rawEntry bin.Address
		// rawBase specifies the base address of a raw binary executable.
		rawBase bin.Address
	)
	flag.Usage = usage
	flag.Var(&blockAddr, "block", "basic block address to disassemble")
	flag.Var(&firstAddr, "first", "first function address to disassemble")
	flag.Var(&funcAddr, "func", "function address to disassemble")
	flag.Var(&lastAddr, "last", "last function address to disassemble")
	flag.BoolVar(&quiet, "q", false, "suppress non-error messages")
	flag.Var(&rawArch, "raw", "machine architecture of raw binary executable (x86_32, x86_64, PowerPC_32, ...)")
	flag.Var(&rawEntry, "rawentry", "entry point of raw binary executable")
	flag.Var(&rawBase, "rawbase", "base address of raw binary executable")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	binPath := flag.Arg(0)
	// Mute debug and warning messages if `-q` is set.
	if quiet {
		dbg.SetOutput(ioutil.Discard)
		warn.SetOutput(ioutil.Discard)
	}

	// Prepare disassembler for the binary executable.
	dis, err := newDisasm(binPath, rawArch, rawEntry, rawBase)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	pretty.Println("dis:", dis)

	// Disassemble basic block.
	if blockAddr != 0 {
		block, err := dis.DecodeBlock(blockAddr)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		_ = block
		return
	}

	// Disassemble function specified by `-func` flag.
	funcAddrs := dis.FuncAddrs
	if funcAddr != 0 {
		funcAddrs = []bin.Address{funcAddr}
	}

	// Disassemble functions.
	for _, funcAddr := range funcAddrs {
		if firstAddr != 0 && funcAddr < firstAddr {
			// skip functions before first address.
			continue
		}
		if lastAddr != 0 && funcAddr >= lastAddr {
			// skip functions after last address.
			break
		}
		f, err := dis.DecodeFunc(funcAddr)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		_ = f
	}
}

// newDisasm returns a new disassembler for the given binary executable.
func newDisasm(binPath string, rawArch bin.Arch, rawEntry, rawBase bin.Address) (*x86.Disasm, error) {
	// Parse raw binary executable.
	if rawArch != 0 {
		file, err := raw.ParseFile(binPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		file.Arch = rawArch
		file.Entry = rawEntry
		file.Segments[0].Addr = rawBase
		return x86.NewDisasm(file)
	}
	// Parse binary executable.
	file, err := bin.ParseFile(binPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return x86.NewDisasm(file)
}
