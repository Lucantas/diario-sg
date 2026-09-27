package mcp

import (
	"context"
	_ "embed"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const sourcesResourceURI = "diario-sg://fontes"

//go:embed fontes.md
var sourcesGuide string

func registerResources(srv *sdk.Server) {
	srv.AddResource(&sdk.Resource{URI: sourcesResourceURI, Name: "fontes", Title: "Fontes do Diário SG", MIMEType: "text/markdown",
		Description: "O que o servidor sabe, de onde vem, que ferramenta devolve cada fonte e o que ficou de fora."},
		func(context.Context, *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: sourcesResourceURI, MIMEType: "text/markdown", Text: sourcesGuide}}}, nil
		})
}
