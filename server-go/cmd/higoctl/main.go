package main

import (
	"flag"
	"fmt"
	"os"

	"higoos/server-go/internal/devstub"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: higoctl <command>\n\nCommands:\n  doctor    Print devstub/mac development diagnostics\n")
	}
	flag.Parse()

	command := "doctor"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "doctor":
		runDoctor()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		flag.Usage()
		os.Exit(2)
	}
}

func runDoctor() {
	report := devstub.NewStore().DoctorReport()
	fmt.Println("HiGoOS backend doctor")
	fmt.Printf("adapter: %s\n", report.Adapter)
	fmt.Printf("host: %s/%s\n", report.HostOS, report.Arch)
	fmt.Printf("booted_at: %s\n", report.BootedAt)
	fmt.Printf("seed_apps: %d\n", report.AppCount)
	fmt.Printf("seed_windows: %d\n", report.WindowCount)
	for _, note := range report.Notes {
		fmt.Printf("note: %s\n", note)
	}
}
