package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/fwojciec/locdoc"
	"google.golang.org/genai"
)

const model = "gemini-2.5-flash"

// Ensure Asker implements locdoc.Asker at compile time.
var _ locdoc.Asker = (*Asker)(nil)

// Asker implements locdoc.Asker using Google Gemini.
type Asker struct {
	client *genai.Client
	docs   locdoc.DocumentService
}

// NewAsker creates a new Asker.
func NewAsker(client *genai.Client, docs locdoc.DocumentService) *Asker {
	return &Asker{client: client, docs: docs}
}

// Ask answers a natural language question about a project's documentation.
func (a *Asker) Ask(ctx context.Context, projectID, question string) (string, error) {
	if projectID == "" {
		return "", locdoc.Errorf(locdoc.EINVALID, "project ID required")
	}
	if question == "" {
		return "", locdoc.Errorf(locdoc.EINVALID, "question required")
	}

	docs, err := a.docs.FindDocuments(ctx, locdoc.DocumentFilter{ProjectID: &projectID})
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "", locdoc.Errorf(locdoc.ENOTFOUND, "no documents found for project %q", projectID)
	}

	prompt := buildPrompt(docs, question)

	result, err := a.client.Models.GenerateContent(ctx, model,
		[]*genai.Content{{
			Parts: []*genai.Part{{Text: prompt}},
		}},
		nil,
	)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", locdoc.Errorf(locdoc.EINTERNAL, "gemini returned nil result")
	}

	return result.Text(), nil
}

func buildPrompt(docs []*locdoc.Document, question string) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant answering questions about software library documentation.\n\n")
	sb.WriteString("<documentation>\n")
	for _, doc := range docs {
		title := doc.Title
		if title == "" {
			title = doc.SourceURL
		}
		fmt.Fprintf(&sb, "## Document: %s\n%s\n\n", title, doc.Content)
	}
	sb.WriteString("</documentation>\n\n")
	fmt.Fprintf(&sb, "Question: %s\n\n", question)
	sb.WriteString("Answer based only on the documentation provided. If the answer is not in the documentation, say so.")
	return sb.String()
}
