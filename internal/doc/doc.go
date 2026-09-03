// Package doc embeds agentforum's help entries (the agent user guide and its
// topic deep-dives) and registers them with the Glazed help system so they are
// queryable straight from the binary:
//
//	agentforum help agent-guide
//	agentforum help configuration
//	agentforum help
//
// Files live in applications/ and topics/ and follow the Glazed help-section
// frontmatter conventions (Title, Slug, Short, Topics, Commands, Flags,
// IsTopLevel, IsTemplate, ShowPerDefault, SectionType).
package doc

import (
	"embed"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *
var docFS embed.FS

// AddDocToHelpSystem loads every embedded help section into helpSystem.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, ".")
}
