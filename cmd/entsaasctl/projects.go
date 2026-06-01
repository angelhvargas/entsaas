package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"entsaas/internal/store"
)

func handleProjectsCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("projects", flag.ExitOnError)
	orgIDOpt := fs.String("org-id", "", "ID of the organization")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	if verb != "list" {
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'projects'\n", verb)
		os.Exit(2)
	}

	if *orgIDOpt == "" {
		fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'list'")
		os.Exit(2)
	}

	runProjectsList(ctx, pgStore, *orgIDOpt)
}

func runProjectsList(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	projects, err := pgStore.ListProjectsByOrg(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch projects: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCREATED AT")

	for _, p := range projects {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID, p.Name, p.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	_ = w.Flush()
	if len(projects) == 0 {
		fmt.Println("No projects found for this organization.")
	}
}
